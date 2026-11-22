package parser

import (
	"lunar/internal/ast"
	"testing"
)

// `speed?: number` was skipped token by token: the field never reached the
// interface, so literals "accepted" it only because it did not exist.
func TestOptionalFieldIsParsed(t *testing.T) {
	statements := parseSource(t, "interface Phase\n    name: string\n    speed?: number\nend\n")
	iface := statements[0].(*ast.InterfaceDeclaration)
	if len(iface.Properties) != 2 {
		t.Fatalf("got %d properties, want 2", len(iface.Properties))
	}
	speed := iface.Properties[1]
	if speed.Name.Value != "speed" || !speed.IsOptional || speed.Type == nil {
		t.Fatalf("got %+v, want optional 'speed' with a type", speed)
	}
	if iface.Properties[0].IsOptional {
		t.Error("'name' should not be optional")
	}
}

func TestOptionalFieldInClassAndTypeShape(t *testing.T) {
	statements := parseSource(t, "class Mover\n    speed?: number\n    name: string = \"m\"\nend\ntype Shape\n    radius?: number\nend\n")
	class := statements[0].(*ast.ClassDeclaration)
	if len(class.Properties) != 2 || !class.Properties[0].IsOptional {
		t.Fatalf("class properties: %+v", class.Properties)
	}
	shape := statements[1].(*ast.TypeDeclaration)
	if len(shape.Properties) != 1 || !shape.Properties[0].IsOptional {
		t.Fatalf("type shape properties: %+v", shape.Properties)
	}
}
