// Package selfupdater implements logic behind self-updating App. It uses github releases to update your app.
package selfupdater

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"runtime"
	"slices"
	"strings"

	"github.com/blang/semver"
	"github.com/google/go-github/v59/github"
)

type repositoryInfo struct {
	ctx      context.Context
	gclient  *github.Client
	assets   []*github.ReleaseAsset
	platform string
}

type installInfo struct {
	assetID   int64
	assetName string
	tmpPath   string
	exePath   string
}

// Updater is the main structure in charge to check latest version and update your app.
type Updater struct {
	Owner   string
	Repo    string
	Current semver.Version
	repositoryInfo
	installInfo
}

// UpdaterOpts represent an option you can pass to [Updater] constructor.
// It uses functional option pattern so the underlying type is `func(*Updater)`.
type UpdaterOpts func(*Updater)

// WithContext will pass the given context to an [Updater] instance.
func WithContext(ctx context.Context) UpdaterOpts {
	return func(u *Updater) {
		u.ctx = ctx
	}
}

// WithHttpClient will pass the given *http.Client to an [Updater] instance.
func WithHttpClient(client *http.Client) UpdaterOpts {
	return func(u *Updater) {
		u.gclient = github.NewClient(client)
	}
}

// New creates a new instance of Updater.
// It needs the owner and repo name to work and the current version of your app (in semver format ->  [semver package])
// You can pass some options (WithContext, WithHttpClient) so that the updater can fits your need.
// If you don't, the Updater will use context.Background and http.DefaultClient by default.
// [semver package]: https://github.com/blang/semver
func New(owner, repo string, current semver.Version, options ...UpdaterOpts) *Updater {
	u := &Updater{
		Owner:   owner,
		Repo:    repo,
		Current: current,
		repositoryInfo: repositoryInfo{
			ctx:      context.Background(),
			gclient:  github.NewClient(http.DefaultClient),
			platform: fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH),
		},
	}

	for _, optn := range options {
		optn(u)
	}

	return u
}

// CheckLatest will check if the current version is the latest.
// It returns a boolean and an error.
// To avoid wrong behaviour, it returns true if an error is encountered.
func (u *Updater) CheckLatest() (bool, error) {
	rel, _, err := u.gclient.Repositories.GetLatestRelease(u.ctx, u.Owner, u.Repo)
	if err != nil {
		return true, err
	}

	latest, err := semver.Parse(strings.ReplaceAll(rel.GetTagName(), "v", ""))
	if err != nil {
		return true, err
	}

	u.assets = rel.Assets

	return latest.LTE(u.Current), nil
}

func (u *Updater) getAsset() (*github.ReleaseAsset, error) {
	index := slices.IndexFunc(u.assets, func(ra *github.ReleaseAsset) bool {
		return strings.Contains(*ra.Name, u.platform)
	})

	if index == -1 {
		err := fmt.Errorf("release asset not found")
		return nil, err
	}

	return u.assets[index], nil
}

func (u *Updater) downloadAsset() error {
	reader, redirect, err := u.gclient.Repositories.DownloadReleaseAsset(u.ctx, u.Owner, u.Repo, u.assetID, u.gclient.Client())
	if err != nil {
		err = fmt.Errorf("failed to download release asset -> %w", err)
		return err
	}

	if redirect != "" {
		return fmt.Errorf("failed to handle redirect url")
	}

	u.tmpPath = path.Join(os.TempDir(), u.assetName)

	f, err := os.Create(u.tmpPath)
	if err != nil {
		err = fmt.Errorf("failed to create temp downloaded release asset -> %w", err)
		return err
	}

	defer func() {
		f.Close()
		reader.Close()
	}()

	_, err = io.Copy(f, reader)
	if err != nil {
		err = fmt.Errorf("failed to write downloaded release asset -> %w", err)
		return err
	}
	return nil
}

func (u *Updater) rollack() error {
	errRem := os.Remove(u.exePath)
	if errRem != nil {
		errRem = fmt.Errorf("failed to remove the new downloaded binary -> %w", errRem)
	}
	errRen := os.Rename(fmt.Sprintf("%s-old", u.exePath), u.exePath)
	if errRen != nil {
		errRen = fmt.Errorf("failed to rename back the old binary -> %w", errRen)
	}

	return errors.Join(errRem, errRen)
}

func (u *Updater) installNewRelease() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to retrieve current executable path -> %w", err)
	}
	u.exePath = exePath

	err = os.Rename(exePath, fmt.Sprintf("%s-old", exePath))
	if err != nil {
		return fmt.Errorf("failed to rename the old binary -> %w", err)
	}

	err = os.Rename(u.tmpPath, exePath)
	if err != nil {
		return fmt.Errorf("failed to rename the new binary with the old name -> %w", err)
	}
	if strings.Contains(u.platform, "linux") {
		err = os.Chmod(exePath, 0775)
		if err != nil {
			return fmt.Errorf("failed to add executable permission on binary -> %w", err)
		}
	}
	err = exec.Command(exePath).Run()
	if err != nil {
		errRoll := u.rollack()
		return fmt.Errorf("failed to rollback (%w) after unsuccessful try on launching new binary -> %w", errRoll, err)
	}

	return nil
}

// Update will perfom the update process which means :
// 1. Retrieve the corresponding asset (based on platform - os/arch - it needs to appear in the name like `my-super-app_linux-amd64`).
// 2. Download latest release asset for the current platform (os/arch).
// 3. Rename the current process executable with a `-old` suffix.
// 4. Give execution permission to the new executable.
// 5. Try to launch the new executable.
// 6. Try to rollack if it fails by removing the download executable and remove the `-old` suffix.
func (u *Updater) Update() error {
	asset, err := u.getAsset()
	if err != nil {
		return err
	}

	u.assetID = asset.GetID()
	u.assetName = asset.GetName()

	err = u.downloadAsset()
	if err != nil {
		return err
	}

	return u.installNewRelease()
}

// CheckAndUpdate will perform both the [Updater.CheckLatest] and [Updater.Update] actions.
// It may seems a better solution for the developper as you don't have to do some plumbering but it enforce the user to update the application.
func (u *Updater) CheckAndUpdate() error {
	isLatest, err := u.CheckLatest()
	if err != nil {
		return err
	}

	if isLatest {
		return nil
	}

	return u.Update()
}
