package types

import (
	"strings"
	"testing"
	"time"
)

// Checking time grows with a literal's size, not exponentially with its
// depth: a node reachable two ways was walked twice at every level.
func TestDeepTableLiteralsCheckQuickly(t *testing.T) {
	src := "local t = " + strings.Repeat("{ a = ", 40) + "1" + strings.Repeat(" }", 40) + "\nreturn t\n"
	start := time.Now()
	for _, err := range checkSource(t, src) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Errorf("checking a 40-deep literal took %v", d)
	}
}
