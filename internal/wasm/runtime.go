package wasm

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Runtime provides WebAssembly runtime support
type Runtime struct {
	modules map[string]*ModuleInstance
}

// ModuleInstance represents an instantiated WASM module
type ModuleInstance struct {
	Name    string
	Exports map[string]interface{}
	Memory  *Memory
}

// Memory represents WebAssembly linear memory
type Memory struct {
	InitialPages int
	MaximumPages int
	Buffer       []byte
}

// NewRuntime creates a new WASM runtime
func NewRuntime() *Runtime {
	return &Runtime{
		modules: make(map[string]*ModuleInstance),
	}
}

// GenerateBrowserRuntime generates JavaScript runtime code for browser
func GenerateBrowserRuntime() string {
	return `
/**
 * Lunar WebAssembly Runtime
 * Provides a complete runtime environment for Lunar code in the browser
 */
class LunarRuntime {
  constructor(options = {}) {
    this.options = {
      memoryInitial: options.memoryInitial || 256,
      memoryMaximum: options.memoryMaximum || 16384,
      enableDebug: options.enableDebug || false,
      ...options
    };

    this.modules = new Map();
    this.memory = null;
    this.luaState = null;
    this.textDecoder = new TextDecoder();
    this.textEncoder = new TextEncoder();
  }

  /**
   * Initialize the runtime
   */
  async init() {
    // Create shared memory
    this.memory = new WebAssembly.Memory({
      initial: this.options.memoryInitial,
      maximum: this.options.memoryMaximum,
      shared: false
    });

    if (this.options.enableDebug) {
      console.log('[Lunar] Runtime initialized', {
        memoryInitial: this.options.memoryInitial,
        memoryMaximum: this.options.memoryMaximum
      });
    }

    return this;
  }

  /**
   * Load a WASM module
   */
  async loadModule(name, wasmPath) {
    try {
      const response = await fetch(wasmPath);
      if (!response.ok) {
        throw new Error(\`Failed to fetch WASM module: \${response.statusText}\`);
      }

      const buffer = await response.arrayBuffer();
      const module = await this.instantiateModule(buffer, name);

      this.modules.set(name, module);

      if (this.options.enableDebug) {
        console.log(\`[Lunar] Module loaded: \${name}\`, module);
      }

      return module;
    } catch (err) {
      console.error(\`[Lunar] Failed to load module \${name}:\`, err);
      throw err;
    }
  }

  /**
   * Instantiate WASM module with imports
   */
  async instantiateModule(buffer, name) {
    const imports = this.createImports();

    const result = await WebAssembly.instantiate(buffer, imports);

    const module = {
      name: name,
      instance: result.instance,
      exports: result.instance.exports,
      memory: this.memory
    };

    // Initialize Lua state if available
    if (module.exports.lunar_newstate) {
      this.luaState = module.exports.lunar_newstate();
    } else if (module.exports.luaL_newstate) {
      this.luaState = module.exports.luaL_newstate();

      // Open standard libraries
      if (module.exports.luaL_openlibs) {
        module.exports.luaL_openlibs(this.luaState);
      }
    }

    return module;
  }

  /**
   * Create WASM imports object
   */
  createImports() {
    const runtime = this;

    return {
      env: {
        memory: this.memory,
        __memory_base: 0,
        __table_base: 0,

        // Standard C library functions
        abort: () => {
          throw new Error('WASM abort()');
        },

        // Console output
        print: (ptr) => {
          const str = runtime.readString(ptr);
          console.log(str);
        },

        puts: (ptr) => {
          const str = runtime.readString(ptr);
          console.log(str);
          return 1;
        },

        printf: (formatPtr, ...args) => {
          const format = runtime.readString(formatPtr);
          console.log(format, ...args);
          return format.length;
        },

        // Error handling
        error: (ptr) => {
          const msg = runtime.readString(ptr);
          throw new Error(msg);
        }
      },

      wasi_snapshot_preview1: {
        // WASI stubs for compatibility
        fd_write: () => 0,
        fd_close: () => 0,
        fd_seek: () => 0,
        fd_read: () => 0,
        proc_exit: (code) => {
          console.log(\`Process exited with code: \${code}\`);
        }
      }
    };
  }

  /**
   * Read null-terminated string from WASM memory
   */
  readString(ptr) {
    if (!this.memory) return '';

    const buffer = new Uint8Array(this.memory.buffer);
    let end = ptr;

    while (buffer[end] !== 0) {
      end++;
      if (end >= buffer.length) break;
    }

    const bytes = buffer.slice(ptr, end);
    return this.textDecoder.decode(bytes);
  }

  /**
   * Write string to WASM memory
   */
  writeString(str, ptr) {
    if (!this.memory) return 0;

    const bytes = this.textEncoder.encode(str + '\0');
    const buffer = new Uint8Array(this.memory.buffer);

    for (let i = 0; i < bytes.length; i++) {
      buffer[ptr + i] = bytes[i];
    }

    return bytes.length;
  }

  /**
   * Execute Lua code
   */
  execute(code, moduleName = 'default') {
    const module = this.modules.get(moduleName);
    if (!module) {
      throw new Error(\`Module not found: \${moduleName}\`);
    }

    // Allocate memory for code string
    const codeLength = code.length + 1;
    let codePtr = 0;

    if (module.exports.malloc) {
      codePtr = module.exports.malloc(codeLength);
      this.writeString(code, codePtr);
    }

    try {
      let result;

      if (module.exports.luaL_dostring) {
        result = module.exports.luaL_dostring(this.luaState, codePtr);
      } else if (module.exports.lunar_execute) {
        result = module.exports.lunar_execute(codePtr);
      } else {
        throw new Error('No execution function available');
      }

      if (result !== 0) {
        // Get error message if available
        if (module.exports.lua_tostring) {
          const errPtr = module.exports.lua_tostring(this.luaState, -1);
          const errMsg = this.readString(errPtr);
          throw new Error(\`Lua execution error: \${errMsg}\`);
        }
        throw new Error(\`Lua execution failed with code: \${result}\`);
      }

      return result;
    } finally {
      // Free allocated memory
      if (codePtr && module.exports.free) {
        module.exports.free(codePtr);
      }
    }
  }

  /**
   * Call Lua function
   */
  call(functionName, args = [], moduleName = 'default') {
    const module = this.modules.get(moduleName);
    if (!module) {
      throw new Error(\`Module not found: \${moduleName}\`);
    }

    if (!module.exports.lunar_call && !module.exports.lua_getglobal) {
      throw new Error('Function calling not supported');
    }

    // Allocate memory for function name
    const namePtr = module.exports.malloc(functionName.length + 1);
    this.writeString(functionName, namePtr);

    try {
      // Get function from global
      if (module.exports.lua_getglobal) {
        module.exports.lua_getglobal(this.luaState, namePtr);
      }

      // Push arguments
      for (const arg of args) {
        this.pushValue(arg, module);
      }

      // Call function
      const nargs = args.length;
      const nresults = 1;
      const result = module.exports.lua_pcall(this.luaState, nargs, nresults, 0);

      if (result !== 0) {
        const errPtr = module.exports.lua_tostring(this.luaState, -1);
        const errMsg = this.readString(errPtr);
        throw new Error(\`Function call failed: \${errMsg}\`);
      }

      // Get return value
      return this.popValue(module);
    } finally {
      if (module.exports.free) {
        module.exports.free(namePtr);
      }
    }
  }

  /**
   * Push JavaScript value to Lua stack
   */
  pushValue(value, module) {
    const exports = module.exports;

    if (typeof value === 'number') {
      exports.lua_pushnumber(this.luaState, value);
    } else if (typeof value === 'string') {
      const ptr = exports.malloc(value.length + 1);
      this.writeString(value, ptr);
      exports.lua_pushstring(this.luaState, ptr);
      exports.free(ptr);
    } else if (typeof value === 'boolean') {
      exports.lua_pushboolean(this.luaState, value ? 1 : 0);
    } else if (value === null || value === undefined) {
      exports.lua_pushnil(this.luaState);
    } else {
      throw new Error(\`Unsupported value type: \${typeof value}\`);
    }
  }

  /**
   * Pop value from Lua stack
   */
  popValue(module) {
    const exports = module.exports;
    const type = exports.lua_type(this.luaState, -1);

    let value;

    switch (type) {
      case 1: // LUA_TBOOLEAN
        value = exports.lua_toboolean(this.luaState, -1) !== 0;
        break;
      case 3: // LUA_TNUMBER
        value = exports.lua_tonumber(this.luaState, -1);
        break;
      case 4: // LUA_TSTRING
        const ptr = exports.lua_tostring(this.luaState, -1);
        value = this.readString(ptr);
        break;
      default:
        value = null;
    }

    exports.lua_pop(this.luaState, 1);
    return value;
  }

  /**
   * Get module exports
   */
  getExports(moduleName = 'default') {
    const module = this.modules.get(moduleName);
    return module ? module.exports : null;
  }

  /**
   * Cleanup and free resources
   */
  destroy() {
    for (const [name, module] of this.modules) {
      if (module.exports.lunar_close) {
        module.exports.lunar_close(this.luaState);
      } else if (module.exports.lua_close) {
        module.exports.lua_close(this.luaState);
      }
    }

    this.modules.clear();
    this.luaState = null;
    this.memory = null;
  }
}

// Export for browser and Node.js
if (typeof module !== 'undefined' && module.exports) {
  module.exports = LunarRuntime;
} else if (typeof window !== 'undefined') {
  window.LunarRuntime = LunarRuntime;
}
`
}

// GenerateNodeRuntime generates Node.js runtime code
func GenerateNodeRuntime() string {
	return `
/**
 * Lunar WebAssembly Runtime for Node.js
 */
const fs = require('fs');
const path = require('path');

class LunarNodeRuntime {
  constructor(options = {}) {
    this.options = {
      memoryInitial: options.memoryInitial || 256,
      memoryMaximum: options.memoryMaximum || 16384,
      enableDebug: options.enableDebug || false,
      ...options
    };

    this.modules = new Map();
    this.memory = null;
    this.luaState = null;
  }

  async init() {
    this.memory = new WebAssembly.Memory({
      initial: this.options.memoryInitial,
      maximum: this.options.memoryMaximum
    });

    return this;
  }

  async loadModule(name, wasmPath) {
    const buffer = fs.readFileSync(path.resolve(wasmPath));
    const arrayBuffer = buffer.buffer.slice(
      buffer.byteOffset,
      buffer.byteOffset + buffer.byteLength
    );

    const imports = this.createImports();
    const result = await WebAssembly.instantiate(arrayBuffer, imports);

    const module = {
      name: name,
      instance: result.instance,
      exports: result.instance.exports,
      memory: this.memory
    };

    this.modules.set(name, module);
    return module;
  }

  createImports() {
    return {
      env: {
        memory: this.memory,
        __memory_base: 0,
        __table_base: 0,
        print: (ptr) => console.log(this.readString(ptr)),
        puts: (ptr) => {
          console.log(this.readString(ptr));
          return 1;
        },
        abort: () => process.exit(1)
      }
    };
  }

  readString(ptr) {
    const buffer = new Uint8Array(this.memory.buffer);
    let end = ptr;
    while (buffer[end] !== 0) end++;
    return Buffer.from(buffer.slice(ptr, end)).toString('utf8');
  }

  execute(code, moduleName = 'default') {
    const module = this.modules.get(moduleName);
    if (!module) {
      throw new Error(\`Module not found: \${moduleName}\`);
    }

    // Implementation similar to browser runtime
    if (module.exports.lunar_execute) {
      return module.exports.lunar_execute(code);
    }

    throw new Error('Execution not supported');
  }
}

module.exports = LunarNodeRuntime;
`
}

// RuntimeConfig represents runtime configuration
type RuntimeConfig struct {
	MemoryInitial int               `json:"memoryInitial"`
	MemoryMaximum int               `json:"memoryMaximum"`
	EnableDebug   bool              `json:"enableDebug"`
	Features      []string          `json:"features"`
	Environment   map[string]string `json:"environment"`
}

// DefaultRuntimeConfig returns default runtime configuration
func DefaultRuntimeConfig() RuntimeConfig {
	return RuntimeConfig{
		MemoryInitial: 256,
		MemoryMaximum: 16384,
		EnableDebug:   false,
		Features:      []string{"mutable-globals", "sign-ext"},
		Environment:   make(map[string]string),
	}
}

// ExportRuntimeConfig exports runtime config as JSON
func ExportRuntimeConfig(config RuntimeConfig) (string, error) {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GeneratePackageJSON generates package.json for Node.js runtime
func GeneratePackageJSON(moduleName string) string {
	return fmt.Sprintf(`{
  "name": "lunar-wasm-%s",
  "version": "1.0.0",
  "description": "Lunar WebAssembly Module: %s",
  "main": "runtime.js",
  "type": "module",
  "scripts": {
    "start": "node example.js",
    "test": "node test.js"
  },
  "keywords": ["lunar", "wasm", "webassembly", "lua"],
  "author": "",
  "license": "MIT",
  "devDependencies": {
    "@types/node": "^20.0.0"
  }
}
`, moduleName, moduleName)
}

// GenerateREADME generates README for WASM module
func GenerateREADME(moduleName string) string {
	return fmt.Sprintf(`# Lunar WASM Module: %s

WebAssembly compilation of Lunar code for browser and Node.js environments.

## Files

- **%s.wasm** - Compiled WebAssembly module
- **%s.js** - JavaScript glue code
- **%s.html** - Browser test page
- **runtime.js** - Complete runtime implementation
- **example.js** - Usage example

## Browser Usage

` + "```html" + `
<!DOCTYPE html>
<html>
<head>
    <script src="runtime.js"></script>
    <script src="%s.js"></script>
</head>
<body>
    <script>
        (async () => {
            const runtime = new LunarRuntime();
            await runtime.init();
            await runtime.loadModule('main', '%s.wasm');

            // Execute Lunar code
            runtime.execute('print("Hello from WASM!")');

            // Call function
            const result = runtime.call('myFunction', [42, "test"]);
            console.log('Result:', result);
        })();
    </script>
</body>
</html>
` + "```" + `

## Node.js Usage

` + "```javascript" + `
const LunarRuntime = require('./runtime.js');

(async () => {
    const runtime = new LunarRuntime();
    await runtime.init();
    await runtime.loadModule('main', './%s.wasm');

    runtime.execute(\`
        function greet(name)
            return "Hello, " .. name
        end
    \`);

    const result = runtime.call('greet', ['World']);
    console.log(result); // "Hello, World"
})();
` + "```" + `

## Performance

The WebAssembly module is optimized for:
- Fast startup time
- Low memory footprint
- Efficient execution

## Browser Compatibility

- Chrome/Edge 57+
- Firefox 52+
- Safari 11+
- Opera 44+

## License

MIT
`, moduleName, moduleName, moduleName, moduleName, moduleName, moduleName, moduleName)
}

// Package represents a complete WASM package
type Package struct {
	Name         string
	WasmFile     string
	JSGlue       string
	HTMLPage     string
	Runtime      string
	PackageJSON  string
	README       string
	Config       RuntimeConfig
}

// CreatePackage creates a complete WASM package
func CreatePackage(moduleName, outputDir string) (*Package, error) {
	pkg := &Package{
		Name:   moduleName,
		Config: DefaultRuntimeConfig(),
	}

	// Generate runtime
	runtimeCode := GenerateBrowserRuntime()
	runtimeFile := filepath.Join(outputDir, "runtime.js")
	if err := os.WriteFile(runtimeFile, []byte(runtimeCode), 0644); err != nil {
		return nil, fmt.Errorf("failed to write runtime: %w", err)
	}
	pkg.Runtime = runtimeFile

	// Generate package.json
	packageJSON := GeneratePackageJSON(moduleName)
	packageFile := filepath.Join(outputDir, "package.json")
	if err := os.WriteFile(packageFile, []byte(packageJSON), 0644); err != nil {
		return nil, fmt.Errorf("failed to write package.json: %w", err)
	}
	pkg.PackageJSON = packageFile

	// Generate README
	readme := GenerateREADME(moduleName)
	readmeFile := filepath.Join(outputDir, "README.md")
	if err := os.WriteFile(readmeFile, []byte(readme), 0644); err != nil {
		return nil, fmt.Errorf("failed to write README: %w", err)
	}
	pkg.README = readmeFile

	return pkg, nil
}
