package lsp

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"lunar/internal/types"
	"strings"
)

// getSignatureHelp provides function signature help at cursor position
func (s *Server) getSignatureHelp(doc *Document, params SignatureHelpParams) *SignatureHelp {
	line := params.Position.Line
	char := params.Position.Character

	// Get the current line text
	lineText := doc.GetLineContent(line)
	if lineText == "" {
		return nil
	}

	// Find the function call context
	funcName, argIndex := s.findFunctionCallContext(lineText, char)
	if funcName == "" {
		return nil
	}

	// Parse and type check to get function type
	l := lexer.New(doc.Content)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		return nil
	}

	// Get stdlib path for type checker
	stdlibPath := getStdlibPathForLSP()
	checker := types.NewChecker(stdlibPath)
	checker.Check(statements)

	// Look up the function type
	env := checker.GetEnv()
	typ, ok := env.Get(funcName)
	if !ok {
		// Try declaration files
		typ, ok = s.declarations.GetDeclaredType(funcName)
		if !ok {
			return nil
		}
	}

	// Convert to function type
	funcType, ok := typ.(*types.FunctionType)
	if !ok {
		return nil
	}

	// Build signature information
	signatures := s.buildSignatureInformation(funcType, funcName)
	if len(signatures) == 0 {
		return nil
	}

	return &SignatureHelp{
		Signatures:      signatures,
		ActiveSignature: 0,
		ActiveParameter: argIndex,
	}
}

// findFunctionCallContext finds the function name and current parameter index
func (s *Server) findFunctionCallContext(lineText string, cursorPos int) (string, int) {
	// Truncate line at cursor position
	if cursorPos > len(lineText) {
		cursorPos = len(lineText)
	}
	text := lineText[:cursorPos]

	// Find the last opening paren
	openParen := strings.LastIndex(text, "(")
	if openParen == -1 {
		return "", 0
	}

	// Count commas after the opening paren to determine parameter index
	afterParen := text[openParen+1:]
	argIndex := 0
	inString := false
	escape := false
	depth := 0

	for _, ch := range afterParen {
		if escape {
			escape = false
			continue
		}

		if ch == '\\' {
			escape = true
			continue
		}

		if ch == '"' || ch == '\'' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		if ch == '(' || ch == '[' || ch == '{' {
			depth++
		} else if ch == ')' || ch == ']' || ch == '}' {
			depth--
		} else if ch == ',' && depth == 0 {
			argIndex++
		}
	}

	// Extract function name before the opening paren
	beforeParen := text[:openParen]
	beforeParen = strings.TrimSpace(beforeParen)

	// Handle method calls (obj:method or obj.method)
	if strings.Contains(beforeParen, ":") {
		parts := strings.Split(beforeParen, ":")
		if len(parts) >= 2 {
			return strings.TrimSpace(parts[len(parts)-1]), argIndex
		}
	}

	if strings.Contains(beforeParen, ".") {
		parts := strings.Split(beforeParen, ".")
		if len(parts) >= 2 {
			return strings.TrimSpace(parts[len(parts)-1]), argIndex
		}
	}

	// Extract just the function name (handle cases like "local x = funcName(")
	words := strings.Fields(beforeParen)
	if len(words) > 0 {
		funcName := words[len(words)-1]
		// Remove any trailing operators
		funcName = strings.TrimRight(funcName, "=:.")
		return funcName, argIndex
	}

	return "", 0
}

// buildSignatureInformation creates signature info from function type
func (s *Server) buildSignatureInformation(funcType *types.FunctionType, funcName string) []SignatureInformation {
	signatures := []SignatureInformation{}

	// Handle overloads
	if len(funcType.Overloads) > 0 {
		for _, overload := range funcType.Overloads {
			sig := s.createSignatureInfo(overload, funcName)
			signatures = append(signatures, sig)
		}
	} else {
		sig := s.createSignatureInfo(funcType, funcName)
		signatures = append(signatures, sig)
	}

	return signatures
}

// createSignatureInfo creates a SignatureInformation from a function type
func (s *Server) createSignatureInfo(funcType *types.FunctionType, funcName string) SignatureInformation {
	// Build parameter list
	params := []ParameterInformation{}
	paramStrings := []string{}

	for i, paramType := range funcType.Parameters {
		paramName := s.getParameterName(i)
		paramTypeStr := types.TypeString(paramType)

		// Check if parameter is optional
		isOptional := i < len(funcType.OptionalParams) && funcType.OptionalParams[i]
		if isOptional {
			paramTypeStr = paramTypeStr + "?"
		}

		paramLabel := paramName + ": " + paramTypeStr
		paramStrings = append(paramStrings, paramLabel)

		params = append(params, ParameterInformation{
			Label:         paramLabel,
			Documentation: "Type: " + paramTypeStr,
		})
	}

	// Handle rest parameters
	if funcType.HasRestParameter {
		paramStrings = append(paramStrings, "...args: any")
		params = append(params, ParameterInformation{
			Label:         "...args: any",
			Documentation: "Rest parameters",
		})
	}

	// Build signature label
	returnTypeStr := types.TypeString(funcType.ReturnType)
	label := funcName + "(" + strings.Join(paramStrings, ", ") + "): " + returnTypeStr

	// Build documentation
	doc := MarkupContent{
		Kind:  "markdown",
		Value: "```lunar\n" + label + "\n```\n\nReturns: `" + returnTypeStr + "`",
	}

	return SignatureInformation{
		Label:         label,
		Documentation: doc,
		Parameters:    params,
	}
}

// getParameterName generates a parameter name based on index
func (s *Server) getParameterName(index int) string {
	paramNames := []string{"arg1", "arg2", "arg3", "arg4", "arg5", "arg6", "arg7", "arg8"}
	if index < len(paramNames) {
		return paramNames[index]
	}
	return "arg" + string(rune('0'+index))
}
