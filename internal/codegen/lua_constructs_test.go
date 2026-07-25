package codegen

import "testing"

func TestLocalFunctionKeepsLocalScope(t *testing.T) {
	output := generate(t, "local function helper(n: number): number\n\treturn n + 1\nend")

	assertContains(t, output, "local function helper(n)")
}

func TestRepeatUntilGeneratesLuaLoop(t *testing.T) {
	output := generate(t, "local i = 0\nrepeat\n\ti = i + 1\nuntil i >= 3")

	assertContains(t, output, "repeat")
	assertContains(t, output, "until i >= 3")
}

// super(...) has to run the parent constructor against the instance being
// built, and must not start a line with '(' or Lua reads it as a call on the
// previous statement.
func TestSuperCallInitialisesParentFields(t *testing.T) {
	output := generate(t, `
class Animal
	protected name: string

	constructor(name: string)
		self.name = name
	end
end

class Dog extends Animal
	constructor(name: string)
		super(name)
	end
end
`)

	assertContains(t, output, "do local __parent = Animal.new(name)")
	assertContains(t, output, "self[__k] = __v")
	assertNotContains(t, output, "\n    (function()")
}
