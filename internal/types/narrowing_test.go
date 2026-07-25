package types

import "testing"

// An optional stays 'T | nil' without narrowing, which makes optional
// parameters unusable: every use has to be guarded, and the guard has to work.
func TestNilCheckNarrowsInThenBranch(t *testing.T) {
	input := `
function use(b: number?): number
	if b ~= nil then
		return b + 1
	end
	return 0
end
`

	for _, err := range checkSource(t, input) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

// The common early-return guard refines the rest of the function.
func TestNilCheckNarrowsAfterEarlyReturn(t *testing.T) {
	input := `
function use(b: number?): number
	if b == nil then
		return 0
	end
	return b * 2
end
`

	for _, err := range checkSource(t, input) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

func TestNilCheckNarrowsInElseBranch(t *testing.T) {
	input := `
function use(b: string?): string
	if b == nil then
		return "none"
	else
		return b
	end
end
`

	for _, err := range checkSource(t, input) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

// Narrowing must not leak: without the guard the optional is still optional.
func TestOptionalIsStillCheckedWithoutAGuard(t *testing.T) {
	errors := checkSource(t, "function use(b: number?): number\n\treturn b + 1\nend")

	if len(errors) == 0 {
		t.Error("expected using an optional without a nil check to fail")
	}
}

// `param: Type?` is the documented optional-parameter syntax, so the argument
// may be omitted.
func TestOptionalTypedParameterMayBeOmitted(t *testing.T) {
	input := `
function withOptional(a: number, b: number?): number
	if b == nil then
		return a
	end
	return a + b
end

local one: number = withOptional(1)
local two: number = withOptional(1, 2)
`

	for _, err := range checkSource(t, input) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

// A call can yield more values than its type records; that is how pcall is used.
func TestMultipleLocalsFromSingleCall(t *testing.T) {
	input := `
function risky(): number
	return 1
end

local ok, result = pcall(risky)
`

	for _, err := range checkSource(t, input) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

// A non-call value still has to match the number of names.
func TestExtraNamesWithoutACallAreStillReported(t *testing.T) {
	errors := checkSource(t, "local a, b = 1")

	if len(errors) == 0 {
		t.Error("expected a mismatch between 2 names and 1 plain value")
	}
}

func TestReadonlyUtilityTypeResolves(t *testing.T) {
	input := `
interface User
	id: number
	name: string
end

type Frozen = Readonly<User>

function show(u: Frozen): string
	return u.name
end
`

	for _, err := range checkSource(t, input) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}
