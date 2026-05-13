// Package version holds build-time version metadata.
package version

// Version is the semantic version, set via ldflags.
var Version = "0.0.0"

// Commit is the full git commit hash, set via ldflags.
var Commit = "0000000000000000000000000000000000000000"

// Date is the build date, set via ldflags.
var Date = "0000-00-00"

// ShortCommit returns the first 12 characters of the commit hash.
func ShortCommit() string {
	if len(Commit) >= 12 {
		return Commit[:12]
	}
	return Commit
}
