package codegen

import "testing"

// Lua allows several targets and values in one assignment.
func TestMultipleAssignment(t *testing.T) {
	output := generate(t, `
local a: number = 1
local b: number = 2
a, b = b, a
`)

	assertContains(t, output, "a, b = b, a")
}

// A single call can fill several targets.
func TestMultipleAssignmentFromCall(t *testing.T) {
	output := generate(t, `
local ok: boolean = false
local err: string = ""
ok, err = pcall(run)
`)

	assertContains(t, output, "ok, err = pcall(run)")
}

// Index and property targets work alongside plain names.
func TestMultipleAssignmentWithComplexTargets(t *testing.T) {
	output := generate(t, `
local t: string[] = {}
local n: number = 0
t[1], n = "x", 1
`)

	assertContains(t, output, `t[1], n = "x", 1`)
}

func TestSingleAssignmentUnchanged(t *testing.T) {
	output := generate(t, `
local a: number = 1
a = 2
`)

	assertContains(t, output, "a = 2")
	assertNotContains(t, output, ", ")
}
