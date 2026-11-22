package codegen

import (
	"lunar/internal/ast"
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"lunar/internal/types"
	"testing"
)

// generateChecked compiles a snippet the way the compiler does: parse, type
// check, generate. Type errors fail the test unless allowErrors is set.
func generateChecked(t *testing.T, input string) string {
	t.Helper()
	p := parser.New(lexer.New(input))
	statements := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	for _, err := range types.NewChecker().Check(statements) {
		t.Errorf("type error: %s", err.Message)
	}
	return New().Generate(statements)
}

const enemySource = `
class Enemy
    hp: number
    constructor(hp: number)
        self.hp = hp
    end
    hit(n: number): void
        self.hp = self.hp - n
    end
end
`

// Enemy.new is a plain function on the class table; Enemy:new passed the
// class as the first argument and failed at run time.
func TestClassNewCompilesToPlainCall(t *testing.T) {
	out := generateChecked(t, enemySource+"local e = Enemy.new(3)\ne.hit(1)\n")
	assertContains(t, out, "local e = Enemy.new(3)")
	assertContains(t, out, "e:hit(1)")
	assertNotContains(t, out, "Enemy:new")
}

// The checker's annotation, not a list of classes seen in this file, decides
// that a call constructs: a class imported from another module has to work.
func TestConstructorCallFollowsCheckerAnnotation(t *testing.T) {
	p := parser.New(lexer.New("local e = Imported(3)\n"))
	statements := p.Parse()
	call := statements[0].(*ast.VariableDeclaration).Values[0].(*ast.CallExpression)
	call.ConstructorCall = true

	assertContains(t, New().Generate(statements), "local e = Imported.new(3)")
}

// A declared class is foreign Lua: calling it is emitted as written.
func TestDeclaredClassCallIsEmittedAsWritten(t *testing.T) {
	out := generateChecked(t, `declare class Sprite
    constructor(path: string)
    draw(): void
end
local s = Sprite("hero.png")
s.draw()
`)
	assertContains(t, out, `local s = Sprite("hero.png")`)
	assertContains(t, out, "s:draw()")
}

// With no type information, a call is emitted as written: '.' stays '.',
// and obj:m() is how the source asks for self.
func TestUnknownReceiverIsEmittedAsWritten(t *testing.T) {
	out := generateChecked(t, `function make(): any
    return {}
end
local c = make()
c.inc()
c:inc()
`)
	assertContains(t, out, "c.inc()")
	assertContains(t, out, "c:inc()")
}

// The plainest Lua module: functions in a table, called with '.'.
func TestLiteralFunctionFieldCallIsPlain(t *testing.T) {
	out := generateChecked(t, `local M = { greet = function(name: string): string
    return "hi " .. name
end }
print(M.greet("bob"))
`)
	assertContains(t, out, `M.greet("bob")`)
	assertNotContains(t, out, "M:greet")
}

// A const string keeps its literal type, and is still a string receiver.
func TestConstStringReceiverUsesColon(t *testing.T) {
	out := generateChecked(t, `const s = "abc"
print(s.upper())
`)
	assertContains(t, out, "s:upper()")
}

// Every class in the union has the method, so the call takes self.
func TestUnionOfClassesReceiverUsesColon(t *testing.T) {
	out := generateChecked(t, `class A
    hit(n: number): number
        return n
    end
end
class B
    hit(n: number): number
        return n * 2
    end
end
local function pick(flag: boolean): A | B
    if flag then
        return A()
    end
    return B()
end
local x = pick(true)
print(x.hit(3))
`)
	assertContains(t, out, "x:hit(3)")
}

// A class exported from a namespace is a class like any other.
func TestNamespaceClassConstructsAndUsesSelf(t *testing.T) {
	out := generateChecked(t, `namespace Game
    export class Enemy
        hp: number
        constructor(hp: number)
            self.hp = hp
        end
        hit(n: number): number
            self.hp = self.hp - n
            return self.hp
        end
    end
end
local e = Game.Enemy(3)
local f = Game.Enemy.new(4)
print(e.hit(1), f.hit(1))
`)
	assertContains(t, out, "Game.Enemy.new(3)")
	assertContains(t, out, "Game.Enemy.new(4)")
	assertContains(t, out, "e:hit(1)")
}
