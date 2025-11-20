package jit

import (
	"fmt"
	"strings"

	"lunar/internal/ast"
	"lunar/internal/lexer"
)

// CodeGenerator generates JIT-optimized Lua code
type CodeGenerator struct {
	optimizer      *JITOptimizer
	profiling      *ProfilingInstrumentation
	enableProfiling bool
	output         strings.Builder
}

// NewCodeGenerator creates a new JIT code generator
func NewCodeGenerator(hints []Hint, enableProfiling bool) *CodeGenerator {
	var detector *HotPathDetector
	if enableProfiling {
		detector = NewHotPathDetector(1000) // Default threshold
	}

	return &CodeGenerator{
		optimizer:       NewJITOptimizer(hints, detector),
		profiling:       NewProfilingInstrumentation(),
		enableProfiling: enableProfiling,
	}
}

// GeneratePreamble generates JIT setup code
func (g *CodeGenerator) GeneratePreamble(hints []Hint) string {
	var sb strings.Builder

	// Add JIT setup
	sb.WriteString(GenerateJITSetup(hints))

	// Add profiling setup if enabled
	if g.enableProfiling {
		sb.WriteString("\n-- Profiling Setup\n")
		sb.WriteString("local __profiling_enabled = true\n\n")
	}

	return sb.String()
}

// GeneratePostamble generates cleanup code
func (g *CodeGenerator) GeneratePostamble() string {
	if g.enableProfiling {
		return g.profiling.GenerateProfilingReport()
	}
	return ""
}

// WrapFunctionWithJIT wraps a function with JIT directives
func (g *CodeGenerator) WrapFunctionWithJIT(fn *ast.FunctionStatement, body string) string {
	var sb strings.Builder

	// Get JIT directives for this function
	directives := g.optimizer.OptimizeFunction(fn)

	// Add directives before function
	for _, directive := range directives {
		sb.WriteString(directive)
		sb.WriteString("\n")
	}

	// Add profiling instrumentation if enabled
	if g.enableProfiling {
		sb.WriteString(g.profiling.InstrumentFunction(fn.Name.Value))
	}

	// Add function body
	sb.WriteString(body)

	// Add profiling end if enabled
	if g.enableProfiling {
		// Insert before return statement
		sb.WriteString(g.profiling.InstrumentFunctionEnd(fn.Name.Value))
	}

	return sb.String()
}

// AnnotationExtractor extracts JIT annotations from source comments
type AnnotationExtractor struct {
	parser   *HintParser
	comments []Comment
}

// Comment represents a source code comment
type Comment struct {
	Text     string
	Position ast.Position
}

// NewAnnotationExtractor creates a new annotation extractor
func NewAnnotationExtractor() *AnnotationExtractor {
	return &AnnotationExtractor{
		parser:   NewHintParser(),
		comments: make([]Comment, 0),
	}
}

// ExtractFromTokens extracts annotations from lexer tokens
func (e *AnnotationExtractor) ExtractFromTokens(tokens []lexer.Token) []Hint {
	for _, token := range tokens {
		if token.Type == lexer.COMMENT {
			comment := Comment{
				Text: token.Literal,
				Position: ast.Position{
					Line:   token.Line,
					Column: token.Column,
				},
			}
			e.comments = append(e.comments, comment)

			// Try to parse as annotation
			e.parser.ParseAnnotation(token.Literal, comment.Position)
		}
	}

	return e.parser.GetHints()
}

// ExtractFromSource extracts annotations from source code
func (e *AnnotationExtractor) ExtractFromSource(source string) []Hint {
	lines := strings.Split(source, "\n")

	for lineNum, line := range lines {
		// Look for comments
		if idx := strings.Index(line, "--"); idx != -1 {
			commentText := strings.TrimSpace(line[idx+2:])
			position := ast.Position{
				Line:   lineNum + 1,
				Column: idx + 1,
			}

			comment := Comment{
				Text:     commentText,
				Position: position,
			}
			e.comments = append(e.comments, comment)

			// Try to parse as annotation
			e.parser.ParseAnnotation(commentText, position)
		}
	}

	return e.parser.GetHints()
}

// OptimizationLevel represents JIT optimization level
type OptimizationLevel int

const (
	OptNone OptimizationLevel = iota
	OptBasic
	OptStandard
	OptAggressive
)

// OptimizationConfig configures JIT optimization behavior
type OptimizationConfig struct {
	Level           OptimizationLevel
	EnableInlining  bool
	EnableUnrolling bool
	EnableTracing   bool
	MaxTraces       int
	MaxRecords      int
	LoopUnrollLimit int
}

// DefaultConfig returns default optimization configuration
func DefaultConfig() OptimizationConfig {
	return OptimizationConfig{
		Level:           OptStandard,
		EnableInlining:  true,
		EnableUnrolling: true,
		EnableTracing:   true,
		MaxTraces:       10000,
		MaxRecords:      40000,
		LoopUnrollLimit: 40,
	}
}

// AggressiveConfig returns aggressive optimization configuration
func AggressiveConfig() OptimizationConfig {
	return OptimizationConfig{
		Level:           OptAggressive,
		EnableInlining:  true,
		EnableUnrolling: true,
		EnableTracing:   true,
		MaxTraces:       20000,
		MaxRecords:      80000,
		LoopUnrollLimit: 100,
	}
}

// GenerateOptimizationCode generates LuaJIT optimization code based on config
func GenerateOptimizationCode(config OptimizationConfig) string {
	var sb strings.Builder

	sb.WriteString("-- LuaJIT Optimization Configuration\n")
	sb.WriteString("local jit = require('jit')\n")
	sb.WriteString("if jit then\n")

	switch config.Level {
	case OptNone:
		sb.WriteString("  jit.off()\n")
	case OptBasic:
		sb.WriteString("  jit.on()\n")
	case OptStandard, OptAggressive:
		sb.WriteString("  jit.on()\n")
		sb.WriteString("  jit.opt.start(\n")

		options := []string{
			fmt.Sprintf("'maxtrace=%d'", config.MaxTraces),
			fmt.Sprintf("'maxrecord=%d'", config.MaxRecords),
			fmt.Sprintf("'loopunroll=%d'", config.LoopUnrollLimit),
		}

		if config.EnableInlining {
			options = append(options, "'hotloop=56'", "'hotexit=10'")
		}

		if config.EnableUnrolling {
			options = append(options, "'instunroll=4'", "'callunroll=3'")
		}

		for i, opt := range options {
			sb.WriteString("    ")
			sb.WriteString(opt)
			if i < len(options)-1 {
				sb.WriteString(",\n")
			} else {
				sb.WriteString("\n")
			}
		}

		sb.WriteString("  )\n")
	}

	sb.WriteString("end\n\n")
	return sb.String()
}

// TraceAnalyzer analyzes JIT traces
type TraceAnalyzer struct {
	traces []TraceInfo
}

// TraceInfo contains information about a JIT trace
type TraceInfo struct {
	ID          int
	FunctionName string
	IRCount     int
	MCode       int
	Aborted     bool
	Reason      string
}

// NewTraceAnalyzer creates a new trace analyzer
func NewTraceAnalyzer() *TraceAnalyzer {
	return &TraceAnalyzer{
		traces: make([]TraceInfo, 0),
	}
}

// GenerateTraceMonitoring generates code to monitor JIT traces
func (a *TraceAnalyzer) GenerateTraceMonitoring() string {
	return `
-- JIT Trace Monitoring
local jit = require('jit')
if jit then
  local v = require('jit.v')
  if v then
    -- Enable verbose mode to see traces
    v.on('/tmp/jit_traces.txt')
  end

  -- Dump module for detailed trace information
  local dump = require('jit.dump')
  if dump then
    dump.on('tbimrsx', '/tmp/jit_dump.txt')
  end
end
`
}

// BailoutAnalyzer analyzes trace bailouts/exits
type BailoutAnalyzer struct {
	bailouts map[string]int
}

// NewBailoutAnalyzer creates a bailout analyzer
func NewBailoutAnalyzer() *BailoutAnalyzer {
	return &BailoutAnalyzer{
		bailouts: make(map[string]int),
	}
}

// RecordBailout records a trace bailout
func (a *BailoutAnalyzer) RecordBailout(location string) {
	a.bailouts[location]++
}

// GetHotBailouts returns bailout locations above threshold
func (a *BailoutAnalyzer) GetHotBailouts(threshold int) []string {
	hot := make([]string, 0)
	for location, count := range a.bailouts {
		if count >= threshold {
			hot = append(hot, fmt.Sprintf("%s (%d bailouts)", location, count))
		}
	}
	return hot
}

// LoopOptimizer optimizes loop structures for JIT
type LoopOptimizer struct {
	unrollThreshold int
}

// NewLoopOptimizer creates a loop optimizer
func NewLoopOptimizer(unrollThreshold int) *LoopOptimizer {
	return &LoopOptimizer{
		unrollThreshold: unrollThreshold,
	}
}

// ShouldUnroll determines if a loop should be unrolled
func (o *LoopOptimizer) ShouldUnroll(iterationCount int) bool {
	return iterationCount > 0 && iterationCount <= o.unrollThreshold
}

// GenerateUnrolledLoop generates unrolled loop code
func (o *LoopOptimizer) GenerateUnrolledLoop(body string, iterations int) string {
	var sb strings.Builder
	sb.WriteString("-- Unrolled loop\n")
	for i := 0; i < iterations; i++ {
		sb.WriteString(body)
		sb.WriteString("\n")
	}
	return sb.String()
}
