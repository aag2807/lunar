# Lunar LSP Setup Guide

This guide will help you set up the Lunar Language Server Protocol (LSP) with Neovim.

## Prerequisites

- Neovim >= 0.9.0
- Go (for building the LSP server)
- Git (for plugin installation)

## Building the LSP Server

The Lunar LSP server is included in the project. Build it using:

```bash
make build
```

This will build three binaries:
- `lunar` - The Lunar compiler
- `lunar2decl` - The declaration generator tool
- `lunar-lsp` - The LSP server

## Installation

### Option 1: Install to System Path

```bash
make install
```

This installs all binaries to `/usr/local/bin` (Linux) or `%USERPROFILE%/bin` (Windows).

### Option 2: Use Local Binary

You can use the LSP server directly from the project directory without installing it system-wide.

## Neovim Configuration

### Automatic Setup

We provide a pre-configured `init.lua` file. To use it:

```bash
# Backup your existing config (if any)
cp ~/.config/nvim/init.lua ~/.config/nvim/init.lua.backup

# Copy the Lunar LSP configuration
cp nvim-lsp-config.lua ~/.config/nvim/init.lua
```

### Manual Setup

If you already have a Neovim configuration, add this to your `init.lua`:

```lua
-- Set up filetype detection for .lunar files
vim.filetype.add({
  extension = {
    lunar = 'lunar',
  },
})

-- Add this to your LSP setup
local lspconfig = require('lspconfig')
local configs = require('lspconfig.configs')

if not configs.lunar then
  configs.lunar = {
    default_config = {
      cmd = { 'lunar-lsp' },  -- or '/path/to/lunar-lsp'
      filetypes = { 'lunar' },
      root_dir = function(fname)
        return lspconfig.util.find_git_ancestor(fname) or vim.fn.getcwd()
      end,
      settings = {},
      single_file_support = true,
    },
  }
end

lspconfig.lunar.setup({
  on_attach = function(client, bufnr)
    -- Your on_attach configuration here
    vim.notify('Lunar LSP attached!', vim.log.levels.INFO)
  end,
})
```

## Required Neovim Plugins

Install these plugins using your favorite plugin manager:

### Using lazy.nvim

```lua
{
  "neovim/nvim-lspconfig",
  "hrsh7th/nvim-cmp",
  "hrsh7th/cmp-nvim-lsp",
  "hrsh7th/cmp-buffer",
  "L3MON4D3/LuaSnip",
}
```

### Using packer.nvim

```lua
use 'neovim/nvim-lspconfig'
use 'hrsh7th/nvim-cmp'
use 'hrsh7th/cmp-nvim-lsp'
use 'hrsh7th/cmp-buffer'
use 'L3MON4D3/LuaSnip'
```

## Testing the Setup

1. **Build and install:**
   ```bash
   make build
   ```

2. **Run the automated test:**
   ```bash
   ./test_lsp_automated.sh
   ```

3. **Test manually in Neovim:**
   ```bash
   nvim examples/class.lunar
   ```

4. **Inside Neovim:**
   - Run `:LspInfo` to check if the LSP is attached
   - Use `gd` to go to definition
   - Use `K` to show hover information
   - Use `gr` to find references
   - Use `<space>rn` to rename symbols

## LSP Features

The Lunar LSP supports:

- ✅ **Syntax diagnostics** - Real-time error checking
- ✅ **Go to definition** - Jump to symbol definitions
- ✅ **Hover information** - Type and documentation on hover
- ✅ **Code completion** - IntelliSense-like completions
- ✅ **Find references** - Find all usages of a symbol
- ✅ **Rename symbols** - Refactor symbol names
- ✅ **Document symbols** - Outline view of file structure
- ✅ **Code actions** - Quick fixes and refactorings
- ✅ **Inlay hints** - Type hints inline
- ✅ **Signature help** - Parameter hints while typing

## Keybindings (Default)

| Key | Action |
|-----|--------|
| `gd` | Go to definition |
| `gD` | Go to declaration |
| `K` | Hover information |
| `gi` | Go to implementation |
| `gr` | Find references |
| `<space>rn` | Rename symbol |
| `<space>ca` | Code actions |
| `<space>f` | Format document |
| `<C-k>` | Signature help |

## Troubleshooting

### LSP not starting

1. Check if the LSP server is in your PATH:
   ```bash
   which lunar-lsp
   ```

2. Test the LSP server manually:
   ```bash
   lunar-lsp --version
   ```

3. Check Neovim logs:
   ```vim
   :lua vim.cmd('e ' .. vim.lsp.get_log_path())
   ```

### LSP attached but no features working

1. Make sure you're editing a `.lunar` file
2. Check if the LSP is actually attached: `:LspInfo`
3. Restart the LSP: `:LspRestart`

### Plugin errors

Make sure all required plugins are installed. Run:
```vim
:Lazy sync  " for lazy.nvim
:PackerSync " for packer.nvim
```

## Additional Resources

- [Neovim LSP Documentation](https://neovim.io/doc/user/lsp.html)
- [nvim-lspconfig](https://github.com/neovim/nvim-lspconfig)
- Lunar Language Documentation

## Support

If you encounter issues:
1. Check the automated test: `./test_lsp_automated.sh`
2. Check Neovim LSP logs: `:lua vim.cmd('e ' .. vim.lsp.get_log_path())`
3. Open an issue on the project repository
