// Package version exposes build information set by the linker.
package version

import "runtime/debug"

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
