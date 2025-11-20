# Testing Lunar LSP with Neovim

This guide explains how to test the Lunar Language Server Protocol (LSP) implementation with Neovim.

## Prerequisites

- Neovim 0.9.5+ installed ✓
- Lunar LSP server built at `/home/user/lunar/lunar-lsp` ✓
- Neovim configuration set up at `~/.config/nvim/init.lua` ✓

## LSP Features Implemented

### 1. **Hover** (`textDocument/hover`)
Shows type information when you hover over a symbol.
- **Keybinding:** `K` (in normal mode)
- **Example:** Place cursor on `myNumber` and press `K` to see its type

### 2. **Go to Definition** (`textDocument/definition`)
Jumps to where a symbol is defined.
- **Keybinding:** `gd` (in normal mode)
- **Example:** Place cursor on `calculate` usage and press `gd` to jump to function definition

### 3. **Find References** (`textDocument/references`)
Finds all places where a symbol is used.
- **Keybinding:** `gr` (in normal mode)
- **Example:** Place cursor on `calculate` and press `gr` to see all usages

### 4. **Rename Symbol** (`textDocument/rename`)
Renames a symbol throughout the document.
- **Keybinding:** `<leader>rn` (in normal mode, leader is usually `\`)
- **Example:** Place cursor on `calculate` and press `\rn`, type new name

### 5. **Code Completion** (`textDocument/completion`)
Provides auto-completion suggestions.
- **Keybinding:** `<C-x><C-o>` (in insert mode)
- **Example:** Start typing and use the keybinding to trigger completion

## Quick Start

### 1. Open the test file in Neovim:

```bash
cd /home/user/lunar
nvim test_lsp.lunar
```

### 2. Verify LSP is running:

In Neovim, run:
```
:LspInfo
```

You should see:
```
Client: lunar-lsp (id: 1, attached to 1 buffers)
	filetypes:       lunar
	cmd:             /home/user/lunar/lunar-lsp
	root directory:  /home/user/lunar
```

### 3. Test each feature:

#### Test Hover (K):
1. Move cursor to line with `local myNumber: number = 42`
2. Press `K` to see type information

#### Test Go to Definition (gd):
1. Move cursor to `calculate(10, 20)` on line 30
2. Press `gd` to jump to the function definition

#### Test Find References (gr):
1. Move cursor to `calculate` function name (line 7)
2. Press `gr` to see all references in quickfix list
3. Navigate with `:cnext` and `:cprev`

#### Test Rename Symbol (\\rn):
1. Move cursor to `calculate` function name
2. Press `\rn` (backslash followed by rn)
3. Type new name (e.g., `compute`)
4. Press Enter
5. All occurrences should be renamed!

#### Test Code Completion (<C-x><C-o>):
1. Enter insert mode (`i`)
2. Start typing a symbol name
3. Press `Ctrl-x Ctrl-o` to trigger completion

## Test File Structure

The `test_lsp.lunar` file contains:
- Variables with type annotations
- A simple function (`calculate`)
- A class with methods (`Calculator`)
- Multiple usages of symbols for testing references

## Troubleshooting

### LSP not starting:
```
:messages
```
Check for error messages

### LSP not responding:
```
:LspRestart
```
Restart the LSP client

### View LSP logs:
The LSP server logs to stderr, which Neovim captures. Check:
```
:lua vim.lsp.set_log_level("debug")
:lua print(vim.lsp.get_log_path())
```

## Manual Testing

If you prefer to test without Neovim, you can use the test script:

```bash
./test_lsp_client.sh
```

This will start the LSP server and show basic connectivity.

## Implementation Details

The Lunar LSP server implements:
- **JSON-RPC 2.0** communication over stdio
- **Full LSP initialization** handshake
- **Document synchronization** (open, change, close, save)
- **Diagnostics** publishing on parse/type errors
- **Hover** with type information from type checker
- **Definition** lookup with AST traversal
- **References** finding with text-based search + word boundaries
- **Rename** using workspace edits
- **Completion** with context-aware suggestions

## Next Steps

Future LSP enhancements could include:
- Document symbols provider
- Workspace symbols provider
- Code actions (quick fixes)
- Semantic tokens
- Multi-file references
- Signature help
- Document formatting
