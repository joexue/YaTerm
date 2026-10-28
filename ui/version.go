package ui

import (
	"runtime/debug"
	"strings"
)

// buildStamp is "<latest git tag>,<short git revision>" of the build, set by
// zz_buildstamp.go, which the Makefile makes for a build and removes after it
var buildStamp string

// stampPart is the version (0) or revision (1) of buildStamp, "" if not set
func stampPart(i int) string {
	parts := strings.SplitN(buildStamp, ",", 2)
	if i < len(parts) {
		return strings.TrimSpace(parts[i])
	}
	return ""
}

// projectURL is the project's page, where its source code is
const projectURL = "https://github.com/joexue/yaterm"

// buildVersion is the version, "unknown" when the build did not set it
func buildVersion() string {
	if v := stampPart(0); v != "" {
		return v
	}
	return "unknown"
}

// buildRevision is the revision, else the one Go puts in a build from a git
// checkout (-buildvcs), with "-dirty" for uncommitted changes, else "unknown"
func buildRevision() string {
	if r := stampPart(1); r != "" {
		return r
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	rev, dirty := "", false
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if rev == "" {
		return "unknown"
	}
	if len(rev) > 7 {
		rev = rev[:7]
	}
	if dirty {
		rev += "-dirty"
	}
	return rev
}
