// Package selfupdater implements logic behind self-updating App. It uses github releases to update your app.
package selfupdater

import (
	"context"
	"net/http"
	"strings"

	"github.com/blang/semver"
	"github.com/google/go-github/v59/github"
)

// Updater is the main structure in charge to check latest version and update your app.
type Updater struct {
	Owner   string
	Repo    string
	Current semver.Version
	ctx     context.Context
	gclient *github.Client
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
		ctx:     context.Background(),
		gclient: github.NewClient(http.DefaultClient),
	}

	for _, optn := range options {
		optn(u)
	}

	return u
}

// CheckLatest will check if the current version is the latest.
// It returns a boolean, the corresponding github release (if there is one) and an error.
// To avoid wrong behaviour, it returns true if an error is encountered.
func (u *Updater) CheckLatest() (bool, *github.RepositoryRelease, error) {
	rel, _, err := u.gclient.Repositories.GetLatestRelease(u.ctx, u.Owner, u.Repo)
	if err != nil {
		return true, nil, err
	}

	latest, err := semver.Parse(strings.ReplaceAll(rel.GetTagName(), "v", ""))
	if err != nil {
		return true, nil, err
	}

	return latest.EQ(u.Current), rel, nil
}
