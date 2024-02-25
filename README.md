# ghr-self-update
A small library to create self-updating app in Go based on github releases


## Usage

1. Download the dependency

`go get github.com/mJehanno/ghr-self-updater`

`go get github.com/blang/semver/v4`

2. Instanciate new Updater

```go
//main.go 
package main

import(
    "context"

    "github.com/blang/semver/v4"
    selfupdate "github.com/mJehanno/ghr-self-updater"
)

func main() {
    ctx := context.Background()
    current = semver.MustParse("1.2.3")
    updater := selfupdate.New("mJehanno", "gtop", current, WithContext(ctx)) // ownerName, repoName, currentVersion, and a bunch of options that are not required.
}
```

3. Check for latest and Update

either in two steps:

```go
//main.go 
package main

import(
    "context"

    "github.com/blang/semver/v4"
    selfupdate "github.com/mJehanno/ghr-self-updater"
)

func main() {
    ctx := context.Background()
    current = semver.MustParse("1.2.3")
    updater := selfupdate.New("mJehanno", "gtop", current, WithContext(ctx)) // ownerName, repoName, currentVersion, and a bunch of options that are not required.

    latest,err := updater.CheckLatest()
    if err != nil {
        // handleErr
    }

    if !latest {
        // here you can ask for user consent for example
        err := updater.Update()
        if err != nil {
            // handleErr
        }
    }
}
```

or in one step:
```go
//main.go 
package main

import(
    "context"

    "github.com/blang/semver/v4"
    selfupdate "github.com/mJehanno/ghr-self-updater"
)

func main() {
    ctx := context.Background()
    current = semver.MustParse("1.2.3")
    updater := selfupdate.New("mJehanno", "gtop", current, WithContext(ctx)) // ownerName, repoName, currentVersion, and a bunch of options that are not required.

    latest,err := updater.CheckAndUpdate()
    if err != nil {
        // handleErr
    }
}
```