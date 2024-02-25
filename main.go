// Package selfupdater implements logic behind self-updating App. It uses github releases to update your app.
package selfupdater

import (
	"context"
	"net/http"

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

type updaterOpts func(*Updater)

// WithContext will pass the given context to an Updater instance.
func WithContext(ctx context.Context) updaterOpts {
	return func(u *Updater) {
		u.ctx = ctx
	}
}

// WithHttpClient will pass the given *http.Client to an Updater instance.
func WithHttpClient(client *http.Client) updaterOpts {
	return func(u *Updater) {
		u.gclient = github.NewClient(client)
	}
}

// New creates a new instance of Updater.
// It needs the owner and repo name to work and the current version of your app (in semver format ->https://github.com/blang/semver )
// You can pass some options (WithContext, WithHttpClient) so that the updater can fits your need.
// If you don't, the Updater will use context.Background and http.DefaultClient by default.
func New(owner, repo string, current semver.Version, options ...updaterOpts) *Updater {
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
