package lsp

import (
	"fmt"
	"lunar/internal/types"
	"sort"
	"strings"
)

// EnhancedCompletionContext holds context for generating completions
type EnhancedCompletionContext struct {
	TriggerChar    string
	ExpectedType   types.Type
	IsInFuncCall   bool
	IsAfterDot     bool
	IsAfterColon   bool
	ObjectName     string
	Query          string
	CursorPosition Position
}

// CompletionItemWithScore wraps a completion item with a relevance score
type CompletionItemWithScore struct {
	Item  CompletionItem
	Score int
}

// enhanceCompletionItem adds rich documentation and formatting to a completion item
func (s *Server) enhanceCompletionItem(item CompletionItem, typ types.Type, context *EnhancedCompletionContext) CompletionItem {
	// Add markdown documentation
	if typ != nil {
		typeStr := types.TypeString(typ)

		// Build documentation based on type
		doc := s.buildTypeDocumentation(typ, item.Label, typeStr)
		item.Documentation = doc

		// Enhance detail field
		item.Detail = typeStr

		// Add snippet for functions
		if funcType, ok := typ.(*types.FunctionType); ok {
			item = s.addFunctionSnippet(item, funcType)
		}
	}

	return item
}

// buildTypeDocumentation creates rich markdown documentation for a type
func (s *Server) buildTypeDocumentation(typ types.Type, name string, typeStr string) MarkupContent {
	var mdBuilder strings.Builder

	mdBuilder.WriteString("```lunar\n")

	switch t := typ.(type) {
	case *types.FunctionType:
		// Function signature
		sig := s.buildFunctionSignature(name, t)
		mdBuilder.WriteString(sig)
		mdBuilder.WriteString("\n```\n\n")

		// Parameter documentation
		if len(t.Parameters) > 0 {
			mdBuilder.WriteString("**Parameters:**\n\n")
			for i, paramType := range t.Parameters {
				paramName := fmt.Sprintf("arg%d", i+1)
				isOptional := i < len(t.OptionalParams) && t.OptionalParams[i]
				optMarker := ""
				if isOptional {
					optMarker = " (optional)"
				}
				mdBuilder.WriteString(fmt.Sprintf("- `%s`: `%s`%s\n",
					paramName, types.TypeString(paramType), optMarker))
			}
			mdBuilder.WriteString("\n")
		}

		// Return type documentation
		mdBuilder.WriteString(fmt.Sprintf("**Returns:** `%s`\n", types.TypeString(t.ReturnType)))

		// Overload information
		if len(t.Overloads) > 0 {
			mdBuilder.WriteString(fmt.Sprintf("\n*+%d overload(s)*\n", len(t.Overloads)))
		}

	case *types.ClassType:
		mdBuilder.WriteString(fmt.Sprintf("class %s\n", name))
		mdBuilder.WriteString("```\n\n")
		mdBuilder.WriteString("**Type:** Class\n\n")

		if len(t.Properties) > 0 {
			mdBuilder.WriteString("**Properties:**\n\n")
			for propName := range t.Properties {
				mdBuilder.WriteString(fmt.Sprintf("- `%s`\n", propName))
			}
		}

	case *types.InterfaceType:
		mdBuilder.WriteString(fmt.Sprintf("interface %s\n", name))
		mdBuilder.WriteString("```\n\n")
		mdBuilder.WriteString("**Type:** Interface\n\n")

		if len(t.Properties) > 0 {
			mdBuilder.WriteString("**Properties:**\n\n")
			for propName := range t.Properties {
				mdBuilder.WriteString(fmt.Sprintf("- `%s`\n", propName))
			}
		}

	default:
		// Simple type
		mdBuilder.WriteString(fmt.Sprintf("%s: %s\n", name, typeStr))
		mdBuilder.WriteString("```\n\n")
		mdBuilder.WriteString(fmt.Sprintf("**Type:** `%s`\n", typeStr))
	}

	return MarkupContent{
		Kind:  "markdown",
		Value: mdBuilder.String(),
	}
}

// buildFunctionSignature creates a function signature string
func (s *Server) buildFunctionSignature(name string, funcType *types.FunctionType) string {
	params := []string{}

	for i, paramType := range funcType.Parameters {
		paramName := fmt.Sprintf("arg%d", i+1)
		paramTypeStr := types.TypeString(paramType)

		isOptional := i < len(funcType.OptionalParams) && funcType.OptionalParams[i]
		if isOptional {
			paramTypeStr += "?"
		}

		params = append(params, fmt.Sprintf("%s: %s", paramName, paramTypeStr))
	}

	if funcType.HasRestParameter {
		params = append(params, "...args: any")
	}

	returnType := types.TypeString(funcType.ReturnType)
	return fmt.Sprintf("function %s(%s): %s", name, strings.Join(params, ", "), returnType)
}

// addFunctionSnippet adds a snippet for function completion
func (s *Server) addFunctionSnippet(item CompletionItem, funcType *types.FunctionType) CompletionItem {
	// Build snippet with tab stops
	snippet := item.Label + "("

	for i := range funcType.Parameters {
		if i > 0 {
			snippet += ", "
		}
		snippet += fmt.Sprintf("${%d:arg%d}", i+1, i+1)
	}

	snippet += ")$0"

	item.InsertText = snippet
	item.InsertTextFormat = SnippetFormat

	return item
}

// scoreCompletionItem assigns a relevance score to a completion item
func (s *Server) scoreCompletionItem(item CompletionItem, context *EnhancedCompletionContext) int {
	score := 0

	// Exact match bonus
	if strings.EqualFold(item.Label, context.Query) {
		score += 1000
	}

	// Prefix match bonus
	if strings.HasPrefix(strings.ToLower(item.Label), strings.ToLower(context.Query)) {
		score += 500
	}

	// Contains match
	if strings.Contains(strings.ToLower(item.Label), strings.ToLower(context.Query)) {
		score += 250
	}

	// Type-based scoring
	switch item.Kind {
	case KeywordCompletion:
		score += 100 // Keywords have medium priority
	case FunctionCompletion, MethodCompletion:
		score += 200 // Functions are high priority
	case VariableCompletion:
		score += 150 // Variables are medium-high priority
	case PropertyCompletion:
		score += 180 // Properties are high priority in member access
	case ClassCompletion, InterfaceKind:
		score += 120 // Types have medium priority
	case SnippetCompletion:
		score += 300 // Snippets are high priority
	}

	// Context-aware scoring
	if context.IsAfterDot || context.IsAfterColon {
		// In member access, properties and methods are more relevant
		if item.Kind == PropertyCompletion || item.Kind == MethodCompletion {
			score += 300
		}
	}

	// Type compatibility bonus (if expected type is available)
	if context.ExpectedType != nil {
		// TODO: Check if item type is assignable to expected type
		// This would require storing the type with each completion item
	}

	// Length penalty (shorter names are slightly preferred)
	score -= len(item.Label) / 10

	return score
}

// sortAndRankCompletions sorts completion items by relevance
func (s *Server) sortAndRankCompletions(items []CompletionItem, context *EnhancedCompletionContext) []CompletionItem {
	// Score all items
	scored := []CompletionItemWithScore{}
	for _, item := range items {
		score := s.scoreCompletionItem(item, context)
		scored = append(scored, CompletionItemWithScore{
			Item:  item,
			Score: score,
		})
	}

	// Sort by score (descending)
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].Score != scored[j].Score {
			return scored[i].Score > scored[j].Score
		}
		// Tie-breaker: alphabetical
		return scored[i].Item.Label < scored[j].Item.Label
	})

	// Extract sorted items and add sortText
	sorted := []CompletionItem{}
	for i, s := range scored {
		item := s.Item
		// Use sortText to maintain order (LSP clients may re-sort)
		item.SortText = fmt.Sprintf("%04d_%s", i, item.Label)

		// Mark top items as preselected
		if i == 0 && s.Score > 500 {
			item.Preselect = true
		}

		sorted = append(sorted, item)
	}

	return sorted
}

// addSnippetCompletions adds common code snippets to completions
func (s *Server) addSnippetCompletions(context *EnhancedCompletionContext) []CompletionItem {
	snippets := []CompletionItem{}

	// Only add snippets when not in member access
	if context.IsAfterDot || context.IsAfterColon {
		return snippets
	}

	// Function snippet
	snippets = append(snippets, CompletionItem{
		Label:            "function",
		Kind:             SnippetCompletion,
		Detail:           "Function declaration",
		InsertText:       "function ${1:name}(${2:params}): ${3:void}\n\t${0}\nend",
		InsertTextFormat: SnippetFormat,
		Documentation: MarkupContent{
			Kind:  "markdown",
			Value: "```lunar\nfunction name(params): returnType\n\t-- body\nend\n```",
		},
	})

	// Class snippet
	snippets = append(snippets, CompletionItem{
		Label:            "class",
		Kind:             SnippetCompletion,
		Detail:           "Class declaration",
		InsertText:       "class ${1:ClassName}\n\t${0}\nend",
		InsertTextFormat: SnippetFormat,
		Documentation: MarkupContent{
			Kind:  "markdown",
			Value: "```lunar\nclass ClassName\n\t-- properties and methods\nend\n```",
		},
	})

	// Interface snippet
	snippets = append(snippets, CompletionItem{
		Label:            "interface",
		Kind:             SnippetCompletion,
		Detail:           "Interface declaration",
		InsertText:       "interface ${1:InterfaceName}\n\t${0}\nend",
		InsertTextFormat: SnippetFormat,
		Documentation: MarkupContent{
			Kind:  "markdown",
			Value: "```lunar\ninterface InterfaceName\n\t-- properties and methods\nend\n```",
		},
	})

	// For loop snippet
	snippets = append(snippets, CompletionItem{
		Label:            "for",
		Kind:             SnippetCompletion,
		Detail:           "For loop",
		InsertText:       "for ${1:i} = ${2:1}, ${3:10} do\n\t${0}\nend",
		InsertTextFormat: SnippetFormat,
		Documentation: MarkupContent{
			Kind:  "markdown",
			Value: "```lunar\nfor i = 1, 10 do\n\t-- body\nend\n```",
		},
	})

	// While loop snippet
	snippets = append(snippets, CompletionItem{
		Label:            "while",
		Kind:             SnippetCompletion,
		Detail:           "While loop",
		InsertText:       "while ${1:condition} do\n\t${0}\nend",
		InsertTextFormat: SnippetFormat,
		Documentation: MarkupContent{
			Kind:  "markdown",
			Value: "```lunar\nwhile condition do\n\t-- body\nend\n```",
		},
	})

	// If statement snippet
	snippets = append(snippets, CompletionItem{
		Label:            "if",
		Kind:             SnippetCompletion,
		Detail:           "If statement",
		InsertText:       "if ${1:condition} then\n\t${0}\nend",
		InsertTextFormat: SnippetFormat,
		Documentation: MarkupContent{
			Kind:  "markdown",
			Value: "```lunar\nif condition then\n\t-- body\nend\n```",
		},
	})

	return snippets
}
