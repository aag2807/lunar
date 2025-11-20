# Lunar LSP Configuration Guide

This guide shows how to configure the Lunar Language Server Protocol (LSP) in various editors and IDEs.

## Table of Contents

1. [Neovim](#neovim)
2. [Visual Studio Code](#visual-studio-code)
3. [Sublime Text](#sublime-text)
4. [Emacs](#emacs)
5. [Helix](#helix)
6. [Kate/KWrite](#katekwrite)
7. [Generic LSP Client](#generic-lsp-client)

---

## Neovim

### Option 1: Minimal Configuration (No Plugins)

See: [`nvim-lsp-config.lua`](./nvim-lsp-config.lua) - Option 1

```lua
-- In ~/.config/nvim/init.lua
vim.api.nvim_create_autocmd({"BufRead", "BufNewFile"}, {
  pattern = "*.lunar",
  callback = function()
    vim.bo.filetype = "lunar"
    local client_id = vim.lsp.start({
      name = 'lunar-lsp',
      cmd = {'lunar-lsp'},  -- Make sure lunar-lsp is in PATH
      root_dir = vim.fn.getcwd(),
    })
  end,
})
```

### Option 2: With nvim-lspconfig

See: [`nvim-lsp-config.lua`](./nvim-lsp-config.lua) - Option 2

**Install nvim-lspconfig:**
```vim
" Using vim-plug
Plug 'neovim/nvim-lspconfig'

" Using lazy.nvim
{ 'neovim/nvim-lspconfig' }
```

**Configure:**
```lua
local lspconfig = require('lspconfig')
local configs = require('lspconfig.configs')

if not configs.lunar_lsp then
  configs.lunar_lsp = {
    default_config = {
      cmd = {'lunar-lsp'},
      filetypes = {'lunar'},
      root_dir = lspconfig.util.root_pattern('.git', 'lunar.config.json'),
    },
  }
end

lspconfig.lunar_lsp.setup{}
```

### Option 3: Full Setup with Autocompletion

See: [`nvim-lsp-config.lua`](./nvim-lsp-config.lua) - Option 3

Includes nvim-cmp for enhanced autocompletion.

### Keybindings

| Key           | Action                  |
|---------------|-------------------------|
| `gd`          | Go to definition        |
| `K`           | Hover documentation     |
| `gr`          | Find references         |
| `<leader>rn`  | Rename symbol           |
| `<leader>ca`  | Code actions            |
| `[d`          | Previous diagnostic     |
| `]d`          | Next diagnostic         |

---

## Visual Studio Code

### Method 1: Using Generic LSP Client

1. Install a generic LSP client extension
2. Configure in `settings.json`:

See: [`vscode-lsp-config.json`](./vscode-lsp-config.json)

```json
{
  "lunar.languageServer": {
    "enabled": true,
    "path": "/path/to/lunar-lsp",
    "args": []
  },
  "files.associations": {
    "*.lunar": "lunar"
  }
}
```

### Method 2: Custom Extension

Create a VSCode extension using the language client.

See: [`vscode-extension.ts`](./vscode-extension.ts)

**Steps:**
1. Create extension with `yo code`
2. Install dependencies: `npm install vscode-languageclient`
3. Use the example code
4. Package and install: `vsce package`

### Keybindings

| Key           | Action                  |
|---------------|-------------------------|
| `F12`         | Go to definition        |
| `Shift+F12`   | Find references         |
| `F2`          | Rename symbol           |
| `Ctrl+Space`  | Trigger completion      |

---

## Sublime Text

### Using LSP Package

1. Install Package Control
2. Install LSP package: `Package Control: Install Package` → `LSP`
3. Configure in `Preferences > Package Settings > LSP > Settings`:

```json
{
  "clients": {
    "lunar": {
      "enabled": true,
      "command": ["/path/to/lunar-lsp"],
      "selector": "source.lunar",
      "languageId": "lunar"
    }
  }
}
```

4. Create syntax definition for `.lunar` files in `Packages/User/Lunar.sublime-syntax`:

```yaml
%YAML 1.2
---
name: Lunar
file_extensions:
  - lunar
scope: source.lunar
```

### Keybindings

Add to `Preferences > Key Bindings`:

```json
[
  { "keys": ["f12"], "command": "lsp_symbol_definition" },
  { "keys": ["shift+f12"], "command": "lsp_symbol_references" },
  { "keys": ["f2"], "command": "lsp_symbol_rename" }
]
```

---

## Emacs

### Using lsp-mode

1. Install `lsp-mode`:

```elisp
(use-package lsp-mode
  :ensure t
  :commands lsp)
```

2. Configure Lunar LSP:

```elisp
;; Add to ~/.emacs or ~/.emacs.d/init.el

(require 'lsp-mode)

;; Define lunar-mode
(define-derived-mode lunar-mode prog-mode "Lunar"
  "Major mode for editing Lunar files.")

(add-to-list 'auto-mode-alist '("\\.lunar\\'" . lunar-mode))

;; Register Lunar LSP client
(lsp-register-client
 (make-lsp-client
  :new-connection (lsp-stdio-connection '("/path/to/lunar-lsp"))
  :major-modes '(lunar-mode)
  :server-id 'lunar-lsp))

;; Enable LSP for lunar-mode
(add-hook 'lunar-mode-hook #'lsp)
```

### Keybindings

| Key       | Action                  |
|-----------|-------------------------|
| `M-.`     | Go to definition        |
| `M-?`     | Find references         |
| `C-c r`   | Rename symbol           |
| `C-c h`   | Hover documentation     |

---

## Helix

### Configuration

Add to `~/.config/helix/languages.toml`:

```toml
[[language]]
name = "lunar"
scope = "source.lunar"
file-types = ["lunar"]
roots = [".git", "lunar.config.json"]
language-server = { command = "lunar-lsp" }
indent = { tab-width = 4, unit = "    " }
```

### Keybindings

Helix uses default LSP keybindings:

| Key       | Action                  |
|-----------|-------------------------|
| `gd`      | Go to definition        |
| `gr`      | Find references         |
| `Space r` | Rename symbol           |
| `K`       | Hover documentation     |

---

## Kate/KWrite

### Configuration

1. Go to `Settings > Configure Kate > LSP Client`
2. Click `User Server Settings`
3. Add:

```json
{
  "servers": {
    "lunar": {
      "command": ["/path/to/lunar-lsp"],
      "root": "",
      "url": "https://github.com/yourusername/lunar",
      "highlightingModeRegex": "^Lunar$"
    }
  }
}
```

4. Create highlighting mode in `~/.local/share/katepart5/syntax/lunar.xml`

---

## Generic LSP Client

### JSON-RPC Communication

The Lunar LSP server uses JSON-RPC 2.0 over stdio.

**Example Initialize Request:**

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "processId": null,
    "rootUri": "file:///path/to/workspace",
    "capabilities": {
      "textDocument": {
        "hover": { "contentFormat": ["markdown", "plaintext"] },
        "definition": { "linkSupport": true },
        "references": {},
        "rename": { "prepareSupport": true },
        "completion": {
          "completionItem": {
            "snippetSupport": true
          }
        }
      }
    }
  }
}
```

**Example Response:**

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "capabilities": {
      "textDocumentSync": 1,
      "hoverProvider": true,
      "definitionProvider": true,
      "referencesProvider": true,
      "renameProvider": true,
      "completionProvider": {
        "triggerCharacters": [".", ":"]
      }
    }
  }
}
```

### Supported LSP Methods

| Method                           | Status | Description                    |
|----------------------------------|--------|--------------------------------|
| `initialize`                     | ✅     | Initialize LSP connection      |
| `initialized`                    | ✅     | Initialization complete        |
| `shutdown`                       | ✅     | Shutdown server                |
| `textDocument/didOpen`           | ✅     | Document opened                |
| `textDocument/didChange`         | ✅     | Document changed               |
| `textDocument/didSave`           | ✅     | Document saved                 |
| `textDocument/didClose`          | ✅     | Document closed                |
| `textDocument/hover`             | ✅     | Hover information              |
| `textDocument/definition`        | ✅     | Go to definition               |
| `textDocument/references`        | ✅     | Find references                |
| `textDocument/rename`            | ✅     | Rename symbol                  |
| `textDocument/completion`        | ✅     | Code completion                |
| `textDocument/publishDiagnostics`| ✅     | Publish diagnostics (errors)   |

---

## Environment Variables

You can configure the LSP server behavior with environment variables:

```bash
# Enable debug logging
LUNAR_LSP_LOG_LEVEL=debug lunar-lsp

# Set custom log file
LUNAR_LSP_LOG_FILE=/tmp/lunar-lsp.log lunar-lsp
```

---

## Troubleshooting

### LSP Not Starting

1. **Check binary path:**
   ```bash
   which lunar-lsp
   # or
   /path/to/lunar-lsp --version
   ```

2. **Check file permissions:**
   ```bash
   chmod +x /path/to/lunar-lsp
   ```

3. **Test manually:**
   ```bash
   echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | lunar-lsp
   ```

### LSP Not Responding

1. **Check logs** - Most editors show LSP logs
2. **Restart LSP client** - Use editor's LSP restart command
3. **Verify file extension** - Make sure file is `.lunar`

### Diagnostics Not Showing

1. Ensure file is saved (some editors only diagnose on save)
2. Check if diagnostics are enabled in editor settings
3. Look for syntax errors in the file

---

## Performance Tips

1. **Disable features you don't use:**
   ```json
   {
     "lunar.hover": { "enable": false },
     "lunar.completion": { "enable": false }
   }
   ```

2. **Increase debounce time** for large files (editor-specific)

3. **Use workspace folders** for better root detection

---

## Contributing

To add support for additional editors:

1. Create a configuration example
2. Test with the LSP server
3. Document keybindings and features
4. Submit a pull request

---

## Resources

- [LSP Specification](https://microsoft.github.io/language-server-protocol/)
- [Lunar Documentation](../README.md)
- [LSP Testing Guide](../LSP_TESTING.md)

---

## Support

For issues or questions:

1. Check the [LSP Testing Guide](../LSP_TESTING.md)
2. Enable debug logging
3. File an issue on GitHub with logs
