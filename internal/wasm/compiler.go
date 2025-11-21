package wasm

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Compiler compiles Lunar code to WebAssembly
type Compiler struct {
	wasmLuaPath    string
	outputDir      string
	optimization   OptimizationLevel
	includeRuntime bool
}

// OptimizationLevel represents WASM optimization level
type OptimizationLevel int

const (
	OptNone OptimizationLevel = iota
	OptSize                    // Optimize for size (-Os)
	OptSpeed                   // Optimize for speed (-O3)
	OptDebug                   // Debug build (-O0)
)

// CompilerOptions configures the WASM compiler
type CompilerOptions struct {
	WasmLuaPath    string            // Path to wasm-lua compiler
	OutputDir      string            // Output directory for WASM files
	Optimization   OptimizationLevel // Optimization level
	IncludeRuntime bool              // Include Lua runtime in WASM
	MemorySize     int               // Initial memory size (pages)
	StackSize      int               // Stack size
	Features       []string          // WASM features to enable
	Exports        []string          // Functions to export
}

// NewCompiler creates a new WASM compiler
func NewCompiler(options CompilerOptions) *Compiler {
	if options.OutputDir == "" {
		options.OutputDir = "./build/wasm"
	}

	if options.WasmLuaPath == "" {
		// Try to find wasm-lua in PATH or use emscripten
		options.WasmLuaPath = "emcc"
	}

	return &Compiler{
		wasmLuaPath:    options.WasmLuaPath,
		outputDir:      options.OutputDir,
		optimization:   options.Optimization,
		includeRuntime: options.IncludeRuntime,
	}
}

// CompileToWasm compiles Lua code to WebAssembly
func (c *Compiler) CompileToWasm(luaCode string, moduleName string) (string, error) {
	// Ensure output directory exists
	if err := os.MkdirAll(c.outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write Lua code to temporary file
	luaFile := filepath.Join(c.outputDir, moduleName+".lua")
	if err := os.WriteFile(luaFile, []byte(luaCode), 0644); err != nil {
		return "", fmt.Errorf("failed to write Lua file: %w", err)
	}

	// Output WASM file
	wasmFile := filepath.Join(c.outputDir, moduleName+".wasm")

	// Compile using emscripten + lua
	if err := c.compileWithEmscripten(luaFile, wasmFile); err != nil {
		return "", err
	}

	return wasmFile, nil
}

// compileWithEmscripten compiles Lua to WASM using Emscripten
func (c *Compiler) compileWithEmscripten(luaFile, wasmFile string) error {
	// Create a C wrapper that embeds the Lua code
	cFile := strings.TrimSuffix(luaFile, ".lua") + ".c"
	if err := c.generateCWrapper(luaFile, cFile); err != nil {
		return err
	}

	// Compile with emscripten
	args := []string{
		cFile,
		"-o", wasmFile,
		"-s", "WASM=1",
		"-s", "EXPORTED_FUNCTIONS=['_main','_luaL_newstate','_lua_close']",
		"-s", "EXPORTED_RUNTIME_METHODS=['ccall','cwrap']",
		"-s", "ALLOW_MEMORY_GROWTH=1",
	}

	// Add optimization flags
	switch c.optimization {
	case OptSize:
		args = append(args, "-Os")
	case OptSpeed:
		args = append(args, "-O3")
	case OptDebug:
		args = append(args, "-O0", "-g")
	default:
		args = append(args, "-O2")
	}

	// Include Lua runtime if needed
	if c.includeRuntime {
		args = append(args, "-llua")
	}

	cmd := exec.Command(c.wasmLuaPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("emscripten compilation failed: %s\n%s", err, stderr.String())
	}

	return nil
}

// generateCWrapper generates C wrapper code for Lua
func (c *Compiler) generateCWrapper(luaFile, cFile string) error {
	// Read Lua code
	luaCode, err := os.ReadFile(luaFile)
	if err != nil {
		return err
	}

	// Generate C wrapper
	wrapper := fmt.Sprintf(`
#include <lua.h>
#include <lualib.h>
#include <lauxlib.h>
#include <emscripten.h>

// Embedded Lua code
static const char* embedded_lua = %s;

EMSCRIPTEN_KEEPALIVE
int main() {
    lua_State* L = luaL_newstate();
    luaL_openlibs(L);

    if (luaL_dostring(L, embedded_lua) != 0) {
        const char* error = lua_tostring(L, -1);
        printf("Error: %%s\n", error);
        lua_close(L);
        return 1;
    }

    lua_close(L);
    return 0;
}

// Export Lua state for calling from JavaScript
EMSCRIPTEN_KEEPALIVE
lua_State* lunar_newstate() {
    lua_State* L = luaL_newstate();
    luaL_openlibs(L);
    luaL_dostring(L, embedded_lua);
    return L;
}

EMSCRIPTEN_KEEPALIVE
void lunar_close(lua_State* L) {
    lua_close(L);
}

EMSCRIPTEN_KEEPALIVE
int lunar_call(lua_State* L, const char* funcname) {
    lua_getglobal(L, funcname);
    if (lua_pcall(L, 0, 1, 0) != 0) {
        return 0;
    }
    return 1;
}
`, quoteString(string(luaCode)))

	return os.WriteFile(cFile, []byte(wrapper), 0644)
}

// quoteString converts a string to C string literal
func quoteString(s string) string {
	var sb strings.Builder
	sb.WriteString("\"")
	for _, ch := range s {
		switch ch {
		case '\n':
			sb.WriteString("\\n")
		case '\r':
			sb.WriteString("\\r")
		case '\t':
			sb.WriteString("\\t")
		case '"':
			sb.WriteString("\\\"")
		case '\\':
			sb.WriteString("\\\\")
		default:
			sb.WriteRune(ch)
		}
	}
	sb.WriteString("\"")
	return sb.String()
}

// WasmModule represents a compiled WASM module
type WasmModule struct {
	Name     string
	Path     string
	Exports  []string
	Size     int64
	JSGlue   string // JavaScript glue code
	HTMLPage string // HTML test page
}

// GenerateJSGlue generates JavaScript glue code for WASM module
func (c *Compiler) GenerateJSGlue(moduleName string) string {
	return fmt.Sprintf(`
// Lunar WASM Module: %s
class LunarWASM {
  constructor() {
    this.module = null;
    this.luaState = null;
  }

  async init() {
    // Load WASM module
    const response = await fetch('%s.wasm');
    const buffer = await response.arrayBuffer();

    const imports = {
      env: {
        memory: new WebAssembly.Memory({ initial: 256 }),
        __memory_base: 0,
        __table_base: 0,
        // Add any required imports here
      }
    };

    const result = await WebAssembly.instantiate(buffer, imports);
    this.module = result.instance;

    // Initialize Lua state
    if (this.module.exports.lunar_newstate) {
      this.luaState = this.module.exports.lunar_newstate();
    }

    return this;
  }

  // Call Lua function from JavaScript
  call(functionName, ...args) {
    if (!this.module || !this.luaState) {
      throw new Error('WASM module not initialized');
    }

    // Convert args and call Lua function
    return this.module.exports.lunar_call(this.luaState, functionName);
  }

  // Execute Lua code
  execute(code) {
    if (!this.module || !this.luaState) {
      throw new Error('WASM module not initialized');
    }

    // Execute Lua code string
    return this.module.exports.luaL_dostring(this.luaState, code);
  }

  // Clean up
  destroy() {
    if (this.module && this.luaState) {
      this.module.exports.lunar_close(this.luaState);
      this.luaState = null;
    }
  }
}

// Create global instance
window.Lunar = new LunarWASM();

// Auto-initialize on page load
window.addEventListener('DOMContentLoaded', async () => {
  try {
    await window.Lunar.init();
    console.log('Lunar WASM module loaded successfully');
  } catch (err) {
    console.error('Failed to load Lunar WASM:', err);
  }
});
`, moduleName, moduleName)
}

// GenerateHTMLPage generates an HTML test page for WASM module
func (c *Compiler) GenerateHTMLPage(moduleName string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Lunar WASM - %s</title>
    <style>
        body {
            font-family: system-ui, -apple-system, sans-serif;
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
            background: #1e1e1e;
            color: #d4d4d4;
        }
        h1 { color: #4ec9b0; }
        .container {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 20px;
            margin-top: 20px;
        }
        .panel {
            background: #252526;
            border: 1px solid #3c3c3c;
            border-radius: 8px;
            padding: 20px;
        }
        textarea {
            width: 100%%;
            height: 300px;
            background: #1e1e1e;
            color: #d4d4d4;
            border: 1px solid #3c3c3c;
            border-radius: 4px;
            padding: 10px;
            font-family: 'Consolas', 'Monaco', monospace;
            font-size: 14px;
        }
        button {
            background: #0e639c;
            color: white;
            border: none;
            padding: 10px 20px;
            border-radius: 4px;
            cursor: pointer;
            font-size: 14px;
            margin-top: 10px;
        }
        button:hover {
            background: #1177bb;
        }
        #output {
            background: #1e1e1e;
            border: 1px solid #3c3c3c;
            border-radius: 4px;
            padding: 15px;
            min-height: 200px;
            font-family: 'Consolas', 'Monaco', monospace;
            font-size: 13px;
            white-space: pre-wrap;
        }
        .success { color: #4ec9b0; }
        .error { color: #f48771; }
        .info { color: #4fc1ff; }
    </style>
</head>
<body>
    <h1>🌙 Lunar WebAssembly Runtime</h1>
    <p>Module: <strong>%s</strong></p>

    <div class="container">
        <div class="panel">
            <h2>Lunar Code</h2>
            <textarea id="code" placeholder="Enter Lunar code here...">-- Example Lunar code
function greet(name: string): string
    return "Hello, " .. name .. "!"
end

print(greet("WASM"))</textarea>
            <button onclick="runCode()">▶️ Run Code</button>
            <button onclick="clearOutput()">🗑️ Clear Output</button>
        </div>

        <div class="panel">
            <h2>Output</h2>
            <div id="output"></div>
        </div>
    </div>

    <script src="%s.js"></script>
    <script>
        // Intercept console.log for output display
        const outputDiv = document.getElementById('output');
        const originalLog = console.log;

        console.log = function(...args) {
            originalLog.apply(console, args);
            const message = args.map(arg =>
                typeof arg === 'object' ? JSON.stringify(arg, null, 2) : String(arg)
            ).join(' ');
            outputDiv.innerHTML += '<div class="info">' + escapeHtml(message) + '</div>';
        };

        console.error = function(...args) {
            const message = args.join(' ');
            outputDiv.innerHTML += '<div class="error">Error: ' + escapeHtml(message) + '</div>';
        };

        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }

        async function runCode() {
            const code = document.getElementById('code').value;
            outputDiv.innerHTML = '';

            try {
                if (!window.Lunar) {
                    throw new Error('Lunar WASM not initialized');
                }

                // Execute code
                const result = await window.Lunar.execute(code);

                if (result === 0) {
                    outputDiv.innerHTML += '<div class="success">✓ Code executed successfully</div>';
                } else {
                    outputDiv.innerHTML += '<div class="error">✗ Execution failed</div>';
                }
            } catch (err) {
                console.error(err.message);
            }
        }

        function clearOutput() {
            outputDiv.innerHTML = '';
        }

        // Show loading message
        console.log('Initializing Lunar WASM runtime...');
    </script>
</body>
</html>
`, moduleName, moduleName, moduleName)
}

// BuildResult contains the result of a WASM build
type BuildResult struct {
	Module   WasmModule
	Success  bool
	Errors   []string
	Warnings []string
	Duration int64 // milliseconds
}

// Build compiles Lunar to WASM and generates all necessary files
func (c *Compiler) Build(luaCode string, moduleName string) (*BuildResult, error) {
	result := &BuildResult{
		Success:  false,
		Errors:   make([]string, 0),
		Warnings: make([]string, 0),
	}

	// Compile to WASM
	wasmPath, err := c.CompileToWasm(luaCode, moduleName)
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		return result, err
	}

	// Get WASM file size
	stat, _ := os.Stat(wasmPath)
	var wasmSize int64
	if stat != nil {
		wasmSize = stat.Size()
	}

	// Generate JavaScript glue code
	jsGlue := c.GenerateJSGlue(moduleName)
	jsFile := filepath.Join(c.outputDir, moduleName+".js")
	if err := os.WriteFile(jsFile, []byte(jsGlue), 0644); err != nil {
		result.Warnings = append(result.Warnings, "Failed to write JS glue: "+err.Error())
	}

	// Generate HTML test page
	htmlPage := c.GenerateHTMLPage(moduleName)
	htmlFile := filepath.Join(c.outputDir, moduleName+".html")
	if err := os.WriteFile(htmlFile, []byte(htmlPage), 0644); err != nil {
		result.Warnings = append(result.Warnings, "Failed to write HTML page: "+err.Error())
	}

	result.Module = WasmModule{
		Name:     moduleName,
		Path:     wasmPath,
		Size:     wasmSize,
		JSGlue:   jsFile,
		HTMLPage: htmlFile,
	}
	result.Success = true

	return result, nil
}
