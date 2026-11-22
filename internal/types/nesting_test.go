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

// A suggestion between equally close names is always the same one.
func TestDidYouMeanIsDeterministic(t *testing.T) {
	for i := 0; i < 20; i++ {
		got := findSimilarNames("cat", []string{"car", "bat", "cap", "hat"}, 1, 1)
		if len(got) != 1 || got[0] != "bat" {
			t.Fatalf("run %d: %v", i, got)
		}
	}
}
