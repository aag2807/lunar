// Package version is the single source of the Lunar version. The compiler,
// the language server and their --version flags all report this value.
package version

import "runtime/debug"

// Version is the release version. Release builds may override it with
//
//	go build -ldflags "-X lunar/internal/version.Version=2.0.1" ./cmd/lunar
var Version = "2.0.0"

// String returns "lunar <version>", followed by the git revision the binary
// was built from when the toolchain recorded one: "lunar 2.0.0 (1a2b3c4d5e6f)".
func String() string {
	if rev := revision(); rev != "" {
		return "lunar " + Version + " (" + rev + ")"
	}
	return "lunar " + Version
}

func revision() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	var rev string
	dirty := false
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if len(rev) > 12 {
		rev = rev[:12]
	}
	if rev != "" && dirty {
		rev += "-dirty"
	}
	return rev
}
