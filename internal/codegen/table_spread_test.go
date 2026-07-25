package codegen

import "testing"

// Spreading records has to copy their keys, not treat them as a sequence.
func TestRecordSpreadCopiesKeys(t *testing.T) {
	output := generate(t, `
local a = {name = "Alice"}
local b = {city = "NYC"}
local merged = {...a, ...b}
`)

	assertContains(t, output, "__temp[__k] = __v")
}

// Array spread keeps its order.
func TestArraySpreadAppendsInOrder(t *testing.T) {
	output := generate(t, `
local a = {1, 2}
local b = {3, 4}
local merged = {...a, ...b}
`)

	assertContains(t, output, "table.insert(__temp, __src[__i])")
}

// The spread source is bound to a local, so a call is not evaluated twice.
func TestSpreadEvaluatesSourceOnce(t *testing.T) {
	output := generate(t, `
local merged = {...makeTable()}
`)

	assertContains(t, output, "local __src = makeTable()")
	if got := countOccurrences(output, "makeTable()"); got != 1 {
		t.Errorf("expected the spread source to be evaluated once, got %d occurrences\n%s", got, output)
	}
}

// Table literal output must not depend on Go map iteration order.
func TestTableLiteralKeyOrderIsStable(t *testing.T) {
	source := `local t = {zebra = 1, alpha = 2, middle = 3, beta = 4}`

	first := generate(t, source)
	for i := 0; i < 8; i++ {
		if got := generate(t, source); got != first {
			t.Fatalf("generated output differs between runs:\n%s\n---\n%s", first, got)
		}
	}
}

// A key that Lua reserves cannot be written bare.
func TestReservedWordKeyIsBracketed(t *testing.T) {
	output := generate(t, `local t = {end_ = 1, name = 2}`)

	assertContains(t, output, "end_ = 1")
	assertContains(t, output, "name = 2")
}

func countOccurrences(haystack, needle string) int {
	count := 0
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			count++
		}
	}
	return count
}
