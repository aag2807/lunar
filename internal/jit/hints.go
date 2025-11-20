package jit

import (
	"fmt"
	"strings"

	"lunar/internal/ast"
)

// JIT compilation hint types for LuaJIT optimization
type HintType int

const (
	HintNone HintType = iota
	HintOn              // Enable JIT compilation for function
	HintOff             // Disable JIT compilation for function
	HintHotPath         // Mark as hot path (frequently executed)
	HintUnroll          // Suggest loop unrolling
	HintInline          // Suggest function inlining
	HintNoInline        // Prevent function inlining
	HintTrace           // Enable trace compilation
	HintNoTrace         // Disable trace compilation
	HintFlush           // Flush JIT cache after function
	HintOptimize        // Aggressive optimization level
)

// Hint represents a JIT compilation hint annotation
type Hint struct {
	Type     HintType
	Target   string            // Function or block name
	Options  map[string]string // Additional options
	Line     int               // Source line
	Column   int               // Source column
}

// HintDirective converts hint to LuaJIT directive
func (h *Hint) HintDirective() string {
	switch h.Type {
	case HintOn:
		if h.Target != "" {
			return fmt.Sprintf("jit.on(%s)", h.Target)
		}
		return "jit.on()"
	case HintOff:
		if h.Target != "" {
			return fmt.Sprintf("jit.off(%s)", h.Target)
		}
		return "jit.off()"
	case HintFlush:
		if h.Target != "" {
			return fmt.Sprintf("jit.flush(%s)", h.Target)
		}
		return "jit.flush()"
	case HintHotPath:
		// Hot paths are automatically detected by LuaJIT,
		// but we can use this for profiling hints
		return fmt.Sprintf("-- HOT PATH: %s", h.Target)
	case HintUnroll:
		// Loop unrolling hint (comment-based for profiling)
		return "-- JIT HINT: unroll"
	case HintInline:
		// Inline hint (comment-based)
		return "-- JIT HINT: inline"
	case HintNoInline:
		return "-- JIT HINT: noinline"
	case HintTrace:
		return "-- JIT HINT: trace"
	case HintNoTrace:
		return "-- JIT HINT: notrace"
	case HintOptimize:
		level := h.Options["level"]
		if level == "" {
			level = "3"
		}
		return fmt.Sprintf("-- JIT OPTIMIZE: level=%s", level)
	default:
		return ""
	}
}

// HintParser parses JIT hint annotations from comments
type HintParser struct {
	hints []Hint
}

// NewHintParser creates a new JIT hint parser
func NewHintParser() *HintParser {
	return &HintParser{
		hints: make([]Hint, 0),
	}
}

// ParseAnnotation parses a JIT hint annotation from a comment
// Format: @jit.on, @jit.off, @hotpath, @inline, @noinline, etc.
func (p *HintParser) ParseAnnotation(comment string, line int, column int) *Hint {
	comment = strings.TrimSpace(comment)
	if !strings.HasPrefix(comment, "@") {
		return nil
	}

	// Remove @ prefix
	annotation := strings.TrimPrefix(comment, "@")
	parts := strings.Fields(annotation)
	if len(parts) == 0 {
		return nil
	}

	hint := &Hint{
		Type:    HintNone,
		Line:    line,
		Column:  column,
		Options: make(map[string]string),
	}

	// Parse hint type
	switch parts[0] {
	case "jit.on", "jit-on":
		hint.Type = HintOn
	case "jit.off", "jit-off":
		hint.Type = HintOff
	case "hotpath", "hot-path":
		hint.Type = HintHotPath
	case "unroll", "loop-unroll":
		hint.Type = HintUnroll
	case "inline":
		hint.Type = HintInline
	case "noinline", "no-inline":
		hint.Type = HintNoInline
	case "trace":
		hint.Type = HintTrace
	case "notrace", "no-trace":
		hint.Type = HintNoTrace
	case "jit.flush", "jit-flush":
		hint.Type = HintFlush
	case "optimize":
		hint.Type = HintOptimize
	default:
		return nil
	}

	// Parse target and options
	if len(parts) > 1 {
		hint.Target = parts[1]
	}

	// Parse key=value options
	for i := 2; i < len(parts); i++ {
		if kv := strings.SplitN(parts[i], "=", 2); len(kv) == 2 {
			hint.Options[kv[0]] = kv[1]
		}
	}

	p.hints = append(p.hints, *hint)
	return hint
}

// GetHints returns all parsed hints
func (p *HintParser) GetHints() []Hint {
	return p.hints
}

// HotPathDetector detects frequently executed code paths
type HotPathDetector struct {
	executionCounts map[string]int64
	threshold       int64
}

// NewHotPathDetector creates a new hot path detector
func NewHotPathDetector(threshold int64) *HotPathDetector {
	return &HotPathDetector{
		executionCounts: make(map[string]int64),
		threshold:       threshold,
	}
}

// RecordExecution records execution of a function or block
func (d *HotPathDetector) RecordExecution(name string, count int64) {
	d.executionCounts[name] += count
}

// IsHotPath checks if a function/block is a hot path
func (d *HotPathDetector) IsHotPath(name string) bool {
	return d.executionCounts[name] >= d.threshold
}

// GetHotPaths returns all detected hot paths
func (d *HotPathDetector) GetHotPaths() []string {
	hotPaths := make([]string, 0)
	for name, count := range d.executionCounts {
		if count >= d.threshold {
			hotPaths = append(hotPaths, name)
		}
	}
	return hotPaths
}

// HotPathReport generates a report of hot paths
type HotPathReport struct {
	HotPaths  []HotPathEntry
	Threshold int64
}

// HotPathEntry represents a hot path entry
type HotPathEntry struct {
	Name           string
	ExecutionCount int64
	Percentage     float64
}

// GenerateReport generates a hot path report
func (d *HotPathDetector) GenerateReport() HotPathReport {
	var totalExecutions int64
	for _, count := range d.executionCounts {
		totalExecutions += count
	}

	entries := make([]HotPathEntry, 0)
	for name, count := range d.executionCounts {
		if count >= d.threshold {
			percentage := float64(count) / float64(totalExecutions) * 100
			entries = append(entries, HotPathEntry{
				Name:           name,
				ExecutionCount: count,
				Percentage:     percentage,
			})
		}
	}

	return HotPathReport{
		HotPaths:  entries,
		Threshold: d.threshold,
	}
}

// JITOptimizer applies JIT hints to AST
type JITOptimizer struct {
	hints    []Hint
	detector *HotPathDetector
}

// NewJITOptimizer creates a new JIT optimizer
func NewJITOptimizer(hints []Hint, detector *HotPathDetector) *JITOptimizer {
	return &JITOptimizer{
		hints:    hints,
		detector: detector,
	}
}

// OptimizeFunction applies JIT optimizations to a function
func (o *JITOptimizer) OptimizeFunction(fn *ast.FunctionDeclaration) []string {
	directives := make([]string, 0)

	// Check for explicit hints
	for _, hint := range o.hints {
		if hint.Target == fn.Name.Value || hint.Target == "" {
			if directive := hint.HintDirective(); directive != "" {
				directives = append(directives, directive)
			}
		}
	}

	// Check for hot path detection
	if o.detector != nil && o.detector.IsHotPath(fn.Name.Value) {
		directives = append(directives, fmt.Sprintf("-- HOT PATH DETECTED: %s", fn.Name.Value))
		directives = append(directives, "jit.on()")
	}

	return directives
}

// GenerateJITSetup generates JIT setup code for Lua output
func GenerateJITSetup(hints []Hint) string {
	var sb strings.Builder

	sb.WriteString("-- JIT Compilation Setup\n")
	sb.WriteString("local jit = require('jit')\n")
	sb.WriteString("if jit then\n")

	// Global JIT settings
	hasGlobalOn := false
	hasGlobalOff := false

	for _, hint := range hints {
		if hint.Target == "" {
			switch hint.Type {
			case HintOn:
				hasGlobalOn = true
			case HintOff:
				hasGlobalOff = true
			}
		}
	}

	if hasGlobalOn {
		sb.WriteString("  jit.on()\n")
	}
	if hasGlobalOff {
		sb.WriteString("  jit.off()\n")
	}

	// JIT configuration
	sb.WriteString("  jit.opt.start(\n")
	sb.WriteString("    'maxtrace=10000',\n")
	sb.WriteString("    'maxrecord=40000',\n")
	sb.WriteString("    'maxirconst=10000',\n")
	sb.WriteString("    'loopunroll=40',\n")
	sb.WriteString("    'instunroll=4',\n")
	sb.WriteString("    'callunroll=3',\n")
	sb.WriteString("    'recunroll=2'\n")
	sb.WriteString("  )\n")
	sb.WriteString("end\n\n")

	return sb.String()
}

// ProfilingInstrumentation adds profiling instrumentation
type ProfilingInstrumentation struct {
	enabled bool
}

// NewProfilingInstrumentation creates profiling instrumentation
func NewProfilingInstrumentation() *ProfilingInstrumentation {
	return &ProfilingInstrumentation{
		enabled: true,
	}
}

// InstrumentFunction adds profiling code to a function
func (p *ProfilingInstrumentation) InstrumentFunction(functionName string) string {
	if !p.enabled {
		return ""
	}

	return fmt.Sprintf(`
-- Profiling for %s
local __prof_%s_start = os.clock()
local __prof_%s_calls = (__prof_%s_calls or 0) + 1
`, functionName, functionName, functionName, functionName)
}

// InstrumentFunctionEnd adds profiling end code
func (p *ProfilingInstrumentation) InstrumentFunctionEnd(functionName string) string {
	if !p.enabled {
		return ""
	}

	return fmt.Sprintf(`
local __prof_%s_end = os.clock()
local __prof_%s_time = (__prof_%s_time or 0) + (__prof_%s_end - __prof_%s_start)
`, functionName, functionName, functionName, functionName, functionName)
}

// GenerateProfilingReport generates code to print profiling report
func (p *ProfilingInstrumentation) GenerateProfilingReport() string {
	return `
-- Print profiling report at program end
local function __print_profiling_report()
  print("\n=== JIT Profiling Report ===")
  for name, time in pairs(_G) do
    if name:match("^__prof_.*_time$") then
      local func_name = name:match("^__prof_(.*)_time$")
      local calls = _G["__prof_" .. func_name .. "_calls"] or 0
      print(string.format("%s: %.6f seconds (%d calls, %.6f sec/call)",
        func_name, time, calls, time/calls))
    end
  end
  print("============================\n")
end

-- Register cleanup
if _G.os and os.exit then
  local old_exit = os.exit
  os.exit = function(...)
    __print_profiling_report()
    old_exit(...)
  end
end
`
}
