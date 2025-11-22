package lsp

import (
	"lunar/internal/types"
	"testing"
)

func TestGetMemberCompletionsFromType_StaticProperties(t *testing.T) {
	// Create an interface type with static and non-static properties
	interfaceType := &types.InterfaceType{
		Name: "GraphicsModule",
		Properties: map[string]types.Type{
			"clear":  &types.FunctionType{Parameters: []types.Type{types.Number, types.Number, types.Number}, ReturnType: types.Void},
			"print":  &types.FunctionType{Parameters: []types.Type{types.String, types.Number, types.Number}, ReturnType: types.Void},
			"width":  types.Number,
			"height": types.Number,
		},
		Methods: map[string]*types.FunctionType{},
		StaticProps: map[string]bool{
			"clear": true,
			"print": true,
		},
	}

	server := &Server{}

	// Test with methodsOnly = true (using : operator)
	// Should only show non-static methods (none in this case)
	methodItems := server.getMemberCompletionsFromType("obj", interfaceType, true)

	if len(methodItems) != 0 {
		t.Errorf("With : operator, expected 0 completions, got %d", len(methodItems))
		for _, item := range methodItems {
			t.Errorf("  Unexpected item: %s", item.Label)
		}
	}

	// Test with methodsOnly = false (using . operator)
	// Should show static functions (clear, print) and non-static properties (width, height)
	dotItems := server.getMemberCompletionsFromType("obj", interfaceType, false)

	expectedDotCount := 4 // clear, print (static functions), width, height (non-static properties)
	if len(dotItems) != expectedDotCount {
		t.Errorf("With . operator, expected %d completions, got %d", expectedDotCount, len(dotItems))
		for _, item := range dotItems {
			t.Logf("  Got item: %s (kind=%d)", item.Label, item.Kind)
		}
	}

	// Verify we have the expected items
	foundItems := make(map[string]bool)
	for _, item := range dotItems {
		foundItems[item.Label] = true
	}

	expectedItems := []string{"clear", "print", "width", "height"}
	for _, expected := range expectedItems {
		if !foundItems[expected] {
			t.Errorf("Expected to find %s in dot completions", expected)
		}
	}
}

func TestGetMemberCompletionsFromType_InstanceMethods(t *testing.T) {
	// Create an interface type with instance methods (non-static)
	interfaceType := &types.InterfaceType{
		Name: "Love",
		Properties: map[string]types.Type{
			"graphics": &types.InterfaceType{Name: "GraphicsModule"},
			"window":   &types.InterfaceType{Name: "WindowModule"},
			"load":     &types.FunctionType{Parameters: []types.Type{}, ReturnType: types.Void},
			"update":   &types.FunctionType{Parameters: []types.Type{types.Number}, ReturnType: types.Void},
			"draw":     &types.FunctionType{Parameters: []types.Type{}, ReturnType: types.Void},
		},
		Methods:     map[string]*types.FunctionType{},
		StaticProps: map[string]bool{}, // No static properties
	}

	server := &Server{}

	// Test with methodsOnly = true (using : operator)
	// Should show instance methods: load, update, draw
	methodItems := server.getMemberCompletionsFromType("love", interfaceType, true)

	expectedMethodCount := 3
	if len(methodItems) != expectedMethodCount {
		t.Errorf("With : operator, expected %d completions, got %d", expectedMethodCount, len(methodItems))
		for _, item := range methodItems {
			t.Logf("  Got method: %s", item.Label)
		}
	}

	// Verify we have the expected methods
	foundMethods := make(map[string]bool)
	for _, item := range methodItems {
		foundMethods[item.Label] = true
	}

	expectedMethods := []string{"load", "update", "draw"}
	for _, expected := range expectedMethods {
		if !foundMethods[expected] {
			t.Errorf("Expected to find method %s in : completions", expected)
		}
	}

	// Test with methodsOnly = false (using . operator)
	// Should show non-function properties: graphics, window
	// (instance methods should not appear with . in this implementation)
	dotItems := server.getMemberCompletionsFromType("love", interfaceType, false)

	// Should show non-function properties only
	foundDotItems := make(map[string]bool)
	for _, item := range dotItems {
		foundDotItems[item.Label] = true
	}

	// Verify non-function properties are present
	if !foundDotItems["graphics"] {
		t.Error("Expected to find graphics in . completions")
	}
	if !foundDotItems["window"] {
		t.Error("Expected to find window in . completions")
	}
}

func TestGetMemberCompletionsFromType_MixedStaticAndInstance(t *testing.T) {
	// Create an interface with both static and instance members
	interfaceType := &types.InterfaceType{
		Name: "MixedModule",
		Properties: map[string]types.Type{
			"create":     &types.FunctionType{Parameters: []types.Type{types.String}, ReturnType: types.Any},
			"destroy":    &types.FunctionType{Parameters: []types.Type{types.Number}, ReturnType: types.Void},
			"getName":    &types.FunctionType{Parameters: []types.Type{}, ReturnType: types.String},
			"setName":    &types.FunctionType{Parameters: []types.Type{types.String}, ReturnType: types.Void},
			"instance":   types.Any,
			"staticData": types.String,
		},
		Methods: map[string]*types.FunctionType{},
		StaticProps: map[string]bool{
			"create":     true,
			"destroy":    true,
			"staticData": true,
		},
	}

	server := &Server{}

	// Test with : operator - should show instance methods only
	methodItems := server.getMemberCompletionsFromType("obj", interfaceType, true)

	foundMethods := make(map[string]bool)
	for _, item := range methodItems {
		foundMethods[item.Label] = true
	}

	// Should find instance methods
	if !foundMethods["getName"] {
		t.Error("Expected to find getName with : operator")
	}
	if !foundMethods["setName"] {
		t.Error("Expected to find setName with : operator")
	}

	// Should NOT find static functions
	if foundMethods["create"] {
		t.Error("Should not find static create with : operator")
	}
	if foundMethods["destroy"] {
		t.Error("Should not find static destroy with : operator")
	}

	// Test with . operator - should show static functions and non-static properties
	dotItems := server.getMemberCompletionsFromType("obj", interfaceType, false)

	foundDotItems := make(map[string]bool)
	for _, item := range dotItems {
		foundDotItems[item.Label] = true
	}

	// Should find static functions
	if !foundDotItems["create"] {
		t.Error("Expected to find static create with . operator")
	}
	if !foundDotItems["destroy"] {
		t.Error("Expected to find static destroy with . operator")
	}

	// Should find non-static property
	if !foundDotItems["instance"] {
		t.Error("Expected to find instance property with . operator")
	}

	// Should NOT find instance methods
	if foundDotItems["getName"] {
		t.Error("Should not find instance method getName with . operator")
	}
	if foundDotItems["setName"] {
		t.Error("Should not find instance method setName with . operator")
	}
}

func TestGetMemberCompletionsFromType_EmptyStaticProps(t *testing.T) {
	// Test with nil StaticProps map (backward compatibility)
	interfaceType := &types.InterfaceType{
		Name: "LegacyModule",
		Properties: map[string]types.Type{
			"method": &types.FunctionType{Parameters: []types.Type{}, ReturnType: types.Void},
			"data":   types.String,
		},
		Methods:     map[string]*types.FunctionType{},
		StaticProps: nil, // Not initialized
	}

	server := &Server{}

	// Should not crash with nil StaticProps
	methodItems := server.getMemberCompletionsFromType("obj", interfaceType, true)

	// All function-type properties should be treated as instance methods
	foundMethods := make(map[string]bool)
	for _, item := range methodItems {
		foundMethods[item.Label] = true
	}

	if !foundMethods["method"] {
		t.Error("Expected to find method with nil StaticProps")
	}

	dotItems := server.getMemberCompletionsFromType("obj", interfaceType, false)

	foundDotItems := make(map[string]bool)
	for _, item := range dotItems {
		foundDotItems[item.Label] = true
	}

	// Non-function properties should appear
	if !foundDotItems["data"] {
		t.Error("Expected to find data property with . operator")
	}
}
