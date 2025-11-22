# LSP Declaration Debugging Guide

This guide helps you diagnose issues with `.d.lunar` declaration files not being detected by the LSP.

## Common Issues and Solutions

### 1. LSP Not Finding .d.lunar Files

**Symptoms:**
- Autocomplete doesn't show declarations from `.d.lunar` files
- Type checking doesn't recognize global types

**Debug Steps:**

1. **Check LSP Logs:**
   The LSP outputs detailed logs to stderr. Look for messages like:
   ```
   [lunar-lsp-decl] Setting root path: /path/to/your/project
   [lunar-lsp-decl] Scanning for .d.lunar files in: /path/to/your/project
   [lunar-lsp-decl] Found .d.lunar file: /path/to/your/project/love.d.lunar
   [lunar-lsp-decl] Loading declaration file: /path/to/your/project/love.d.lunar
   [lunar-lsp-decl] Loaded 3 symbols from /path/to/your/project/love.d.lunar
   [lunar-lsp-decl]   - Color
   [lunar-lsp-decl]   - Love
   [lunar-lsp-decl]   - love
   ```

2. **View LSP Logs in Your Editor:**

   **Neovim:**
   ```lua
   -- In your LSP config
   vim.lsp.set_log_level("debug")
   -- Then check the log file:
   -- :lua print(vim.lsp.get_log_path())
   ```

   **VS Code:**
   - Open Output panel (View → Output)
   - Select "Lunar Language Server" from dropdown

3. **Verify File Location:**
   - `.d.lunar` files should be in your project root or subdirectories
   - The LSP scans from the workspace root
   - Files in `node_modules/`, `.git/`, or hidden directories are skipped

4. **Check File Syntax:**
   Compile your `.d.lunar` file to check for syntax errors:
   ```bash
   lunar your-file.d.lunar
   ```

   If there are parse errors, the file won't be loaded.

### 2. Static Keyword Not Working

**Symptoms:**
- Autocomplete shows wrong methods for `.` vs `:` operators
- Module functions appear when typing `object:`

**Solution:**
Update your `.d.lunar` files to use the `static` keyword for module functions:

**Old Syntax (Incorrect):**
```lunar
declare interface GraphicsModule
    clear: function(r: number, g: number, b: number): void
    print: function(text: string, x: number, y: number): void
end
```

**New Syntax (Correct):**
```lunar
declare interface GraphicsModule
    static clear(r: number, g: number, b: number): void
    static print(text: string, x: number, y: number): void
end
```

### 3. Autocomplete Behavior

**Understanding the difference:**

- **Dot operator (`.`)**: Shows static functions and non-function properties
  - `love.graphics.` → shows `clear`, `print`, `setColor` (static methods)
  - `love.` → shows `graphics`, `window`, `timer` (properties)

- **Colon operator (`:`)**: Shows instance methods only
  - `love:` → shows `load`, `update`, `draw` (instance methods)
  - `love.graphics:` → shows nothing (all methods are static)

## Example: Correct LÖVE2D Declaration File

```lunar
-- Graphics module with static methods
declare interface GraphicsModule
    static clear(r: number, g: number, b: number, a: number): void
    static print(text: string, x: number, y: number): void
    static setColor(r: number, g: number, b: number, a: number): void
end

-- Main LÖVE interface
declare interface Love
    -- Properties (accessed with love.property)
    graphics: GraphicsModule
    window: WindowModule

    -- Instance methods (called with love:method())
    load(): void
    update(dt: number): void
    draw(): void
end

-- Global love object
declare const love: Love
```

**Usage in code:**
```lunar
-- Static method calls (use dot)
love.graphics.clear(0, 0, 0, 1)
love.graphics.print("Hello", 100, 100)

-- Instance method calls (use colon)
love:load()
love:update(0.016)
love:draw()

-- Property access (use dot)
local gfx: GraphicsModule = love.graphics
```

## Testing Your Setup

Create a test file to verify autocomplete works:

```lunar
-- test.lunar

-- Type "love." and you should see: graphics, window, timer, etc.
love.

-- Type "love:" and you should see: load, update, draw, etc.
love:

-- Type "love.graphics." and you should see: clear, print, setColor, etc.
love.graphics.
```

## Still Having Issues?

1. **Restart your LSP server**
   - Neovim: `:LspRestart`
   - VS Code: Reload window

2. **Check workspace root**
   - Ensure your editor is opening the correct directory as workspace root
   - The LSP scans from this root directory

3. **Verify file permissions**
   - Ensure `.d.lunar` files are readable
   - Check file ownership and permissions

4. **Enable verbose logging**
   - The LSP logs will show exactly what files are being scanned
   - Look for parse errors or type errors in the logs

5. **Check for hidden characters**
   - Ensure `.d.lunar` files use UTF-8 encoding
   - Remove any BOM (Byte Order Mark) if present

## Reporting Issues

If problems persist, include the following in your bug report:

1. LSP log output (showing the scan and load messages)
2. Your `.d.lunar` file contents
3. The Lunar version (`lunar --version`)
4. Your editor and LSP client version
5. The directory structure of your project

## Example Debug Session

```bash
# 1. Build the latest LSP
go build -o bin/lunar-lsp ./cmd/lunar-lsp

# 2. Start the LSP manually to see logs
./bin/lunar-lsp 2>&1 | tee lsp-debug.log

# 3. Open your editor in another terminal
# 4. Check lsp-debug.log for:
#    - Root path being set
#    - .d.lunar files being found
#    - Symbols being loaded
#    - Any parse or type errors
```

Look for these patterns in the logs:

✅ **Success:**
```
[lunar-lsp-decl] Setting root path: /home/user/myproject
[lunar-lsp-decl] Found .d.lunar file: /home/user/myproject/love.d.lunar
[lunar-lsp-decl] Loaded 3 symbols from /home/user/myproject/love.d.lunar
```

❌ **Failure (syntax error):**
```
[lunar-lsp-decl] Found .d.lunar file: /home/user/myproject/love.d.lunar
[lunar-lsp-decl] Parse errors in /home/user/myproject/love.d.lunar:
[lunar-lsp-decl]   expected next token to be :, got ( instead
```

❌ **Failure (not found):**
```
[lunar-lsp-decl] Scan complete. Found 0 .d.lunar files
```
