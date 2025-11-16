# Lunar LSP Quick Start Guide

Quick reference for implementing and using the Lunar Language Server.

## For Implementers

### 1. Initial Setup (30 minutes)

```bash
# Create project structure
mkdir -p cmd/lunar-lsp internal/lsp

# Install dependencies
go get github.com/sourcegraph/jsonrpc2
```

### 2. Minimal Working Server (2 hours)

**cmd/lunar-lsp/main.go**:
```go
package main

import (
    "context"
    "lunar/internal/lsp"
    "os"
)

func main() {
    server := lsp.NewServer()
    server.Run(context.Background(), os.Stdin, os.Stdout)
}
```

**internal/lsp/server.go**:
```go
package lsp

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
)

type Server struct {
    documents *DocumentManager
}

func NewServer() *Server {
    return &Server{
        documents: NewDocumentManager(),
    }
}

func (s *Server) Run(ctx context.Context, in io.Reader, out io.Writer) {
    // Read JSON-RPC messages from stdin
    // Process and respond
    // See: docs/LSP_DESIGN.md for full implementation
}
```

### 3. Test It

```bash
# Build
go build -o lunar-lsp ./cmd/lunar-lsp

# Test with echo
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | ./lunar-lsp

# Test with Neovim
nvim test.lunar
```

## For Users

### Quick Neovim Setup (5 minutes)

**1. Install LSP server:**
```bash
cd /path/to/lunar
go build -o lunar-lsp ./cmd/lunar-lsp
sudo cp lunar-lsp /usr/local/bin/
```

**2. Minimal Neovim config** (`~/.config/nvim/init.lua`):
```lua
-- Install nvim-lspconfig first:
-- :Lazy install nvim-lspconfig (if using Lazy.nvim)

local lspconfig = require("lspconfig")

-- Define Lunar LSP
lspconfig.lunar = {
  default_config = {
    cmd = { "lunar-lsp" },
    filetypes = { "lunar" },
    root_dir = lspconfig.util.find_git_ancestor,
  },
}

lspconfig.lunar.setup({})

-- Basic keybindings
vim.keymap.set("n", "gd", vim.lsp.buf.definition)
vim.keymap.set("n", "K", vim.lsp.buf.hover)
```

**3. Detect .lunar files** (`~/.config/nvim/ftdetect/lunar.vim`):
```vim
au BufRead,BufNewFile *.lunar set filetype=lunar
```

**4. Open a Lunar file:**
```bash
nvim examples/todo_app.lunar
```

Done! You should see:
- ✓ Errors underlined
- ✓ `gd` jumps to definition
- ✓ `K` shows type info

## Implementation Roadmap

### Week 1: Foundation
- [x] Design architecture
- [ ] Basic RPC handling
- [ ] Document manager
- [ ] Initialize/shutdown

### Week 2: Diagnostics
- [ ] Parse on file open/change
- [ ] Type check
- [ ] Send diagnostics
- [ ] Test with real files

### Week 3: Core Features
- [ ] Hover (type info)
- [ ] Go to definition
- [ ] Basic completion

### Week 4: Polish
- [ ] Find references
- [ ] Rename
- [ ] Document symbols
- [ ] Performance optimization

## Testing Checklist

- [ ] Server starts without errors
- [ ] Neovim detects .lunar files
- [ ] LSP attaches to buffer (`:LspInfo`)
- [ ] Syntax errors show up
- [ ] Type errors show up
- [ ] Hover shows type information
- [ ] Go to definition works
- [ ] Completion suggests variables
- [ ] No crashes on rapid typing

## Common Issues

**Q: "lunar-lsp: command not found"**
A: Add to PATH or use full path in Neovim config

**Q: "LSP not attaching"**
A: Check filetype with `:set ft?` - should say "lunar"

**Q: "No diagnostics"**
A: Check `:LspInfo` and `~/.local/state/nvim/lsp.log`

**Q: "Server crashes"**
A: Run manually: `lunar-lsp --verbose` to see errors

## Example Test File

Create `test.lunar`:
```lunar
-- This should show error (undefined variable)
local x: number = unknownVar

-- This should work
local y: string = "hello"

-- Hover over this should show "function(a: number, b: number): number"
function add(a: number, b: number): number
    return a + b
end

-- Go to definition should jump to 'add' function
local result: number = add(1, 2)
```

Test:
1. Open in Neovim: `nvim test.lunar`
2. Should see red underline on `unknownVar`
3. Hover over `add` to see signature
4. `gd` on `add(1, 2)` should jump to definition

## Resources

- **Full Design**: [LSP_DESIGN.md](./LSP_DESIGN.md)
- **Neovim Setup**: [NEOVIM_SETUP.md](./NEOVIM_SETUP.md)
- **LSP Spec**: https://microsoft.github.io/language-server-protocol/

## Next Steps

1. Read [LSP_DESIGN.md](./LSP_DESIGN.md) for architecture
2. Implement basic server (Steps 1-4 from design doc)
3. Test with [NEOVIM_SETUP.md](./NEOVIM_SETUP.md) config
4. Iterate and add features!
