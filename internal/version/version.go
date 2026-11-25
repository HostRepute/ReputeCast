// Package version exposes build information set by the linker.
package version

import (
	"regexp"
	"runtime/debug"
)

// Set with -ldflags "-X github.com/hostrepute/reputecast/internal/version.Version=v1.4.2 ...".
var (
	Version = "dev"
	Commit  = ""
	Date    = ""
)

// Info is the build information shown by `reputecast version` and the
// Instance admin › Version & updates page.
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit,omitempty"`
	Date      string `json:"date,omitempty"`
	GoVersion string `json:"goVersion"`
}

// Get returns the build information of the running binary.
func Get() Info {
	info := Info{Version: Version, Commit: Commit, Date: Date}
	if bi, ok := debug.ReadBuildInfo(); ok {
		info.GoVersion = bi.GoVersion
		if info.Commit == "" {
			for _, s := range bi.Settings {
				switch s.Key {
				case "vcs.revision":
					info.Commit = s.Value
				case "vcs.time":
					if info.Date == "" {
						info.Date = s.Value
					}
				}
			}
		}
	}
	return info
}

// releaseTag is a release's version (v1.8.3, v1.9.0-rc.1): a tag of the
// repository.
var releaseTag = regexp.MustCompile(`^v\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$`)

// SourceURL is where the source code of the running build is, in a GitHub
// repository (owner/name): its release tag, else its commit, else the
// repository itself.
func SourceURL(repo string) string {
	u := "https://github.com/" + repo
	i := Get()
	switch {
	case releaseTag.MatchString(i.Version):
		return u + "/tree/" + i.Version
	case i.Commit != "":
		return u + "/tree/" + i.Commit
	}
	return u
}
