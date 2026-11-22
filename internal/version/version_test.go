package version

import (
	"regexp"
	"testing"
)

// The book pins a compiler version and the engine's editor compares against
// `lunar --version`, so the format is part of the contract.
func TestStringIsLunarThenVersionThenOptionalRevision(t *testing.T) {
	got := String()
	want := regexp.MustCompile(`^lunar ` + regexp.QuoteMeta(Version) + `( \([0-9a-f]{7,12}(-dirty)?\))?$`)
	if !want.MatchString(got) {
		t.Fatalf("String() = %q, want it to match %s", got, want)
	}
}

func TestVersionIsSemver(t *testing.T) {
	if !regexp.MustCompile(`^\d+\.\d+\.\d+(-[0-9A-Za-z.]+)?$`).MatchString(Version) {
		t.Fatalf("Version = %q is not semver", Version)
	}
}
