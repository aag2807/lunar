package types

import (
	"lunar/internal/ast"
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"strings"
	"testing"
)

const enemyClass = `
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

// Enemy.new(3) is how Lua code constructs one; it is the constructor.
func TestClassNewIsTheConstructor(t *testing.T) {
	for _, err := range checkSource(t, enemyClass+"local a: Enemy = Enemy.new(3)\nlocal b: Enemy = Enemy(4)\n") {
		t.Errorf("unexpected type error: %s", err.Message)
	}
	errs := checkSource(t, enemyClass+"local a: Enemy = Enemy.new()\n")
	if len(errs) != 1 {
		t.Errorf("expected an argument-count error for Enemy.new(), got %v", messages(errs))
	}
}

func TestNewOnAnInstanceIsAnError(t *testing.T) {
	errs := checkSource(t, enemyClass+"local e: Enemy = Enemy(3)\nlocal f = e.new(4)\n")
	if len(errs) != 1 || errs[0].Message != "'new' is static; call Enemy.new(...) or Enemy(...)" {
		t.Fatalf("got %v", messages(errs))
	}
}

func TestDeclaringNewIsAnError(t *testing.T) {
	errs := checkSource(t, `
class Pool
    static new(): Pool
        return Pool()
    end
end
`)
	if len(errs) != 1 || errs[0].Message != "'new' is reserved for the constructor of class 'Pool'" {
		t.Fatalf("got %v", messages(errs))
	}
}

// A declared class describes foreign Lua; it may well have its own new.
func TestDeclaredClassMayDeclareNew(t *testing.T) {
	for _, err := range checkSource(t, `
declare class Sound
    static new(path: string): Sound
    play(): void
end
local s: Sound = Sound.new("boom.ogg")
s.play()
`) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

func TestUnknownClassMemberIsAnError(t *testing.T) {
	errs := checkSource(t, enemyClass+"local e: Enemy = Enemy(3)\nlocal n: number = e.hpp\n")
	if len(errs) != 1 || errs[0].Message != "Class 'Enemy' has no member 'hpp'. Did you mean 'hp'?" {
		t.Fatalf("got %v", messages(errs))
	}
}

// The checker, not the code generator, decides that a call constructs an
// instance, so classes from other modules construct correctly too.
func TestConstructorCallsAreAnnotated(t *testing.T) {
	src := enemyClass + "local a = Enemy(3)\nlocal b = Enemy.new(3)\n"
	p := parser.New(lexer.New(src))
	statements := p.Parse()
	if errs := NewChecker().Check(statements); len(errs) > 0 {
		t.Fatalf("type errors: %v", messages(errs))
	}
	a := statements[1].(*ast.VariableDeclaration).Values[0].(*ast.CallExpression)
	if !a.ConstructorCall {
		t.Error("Enemy(3) is not marked as a constructor call")
	}
	b := statements[2].(*ast.VariableDeclaration).Values[0].(*ast.CallExpression)
	if b.ConstructorCall {
		t.Error("Enemy.new(3) already names the constructor; it must not be marked again")
	}
	if dot := b.Function.(*ast.DotExpression); dot.MethodDispatch != ast.DispatchPlain {
		t.Errorf("Enemy.new dispatch = %q, want plain '.'", dot.MethodDispatch)
	}
	if !strings.Contains(src, "Enemy.new") {
		t.Fatal("fixture changed")
	}
}
