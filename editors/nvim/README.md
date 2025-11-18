# Lunar Neovim Plugin

LSP integration for the Lunar programming language in Neovim.

## Features

- Real-time diagnostics (type errors, parse errors)
- Hover information with type details
- Go to definition
- Auto-completion
- Syntax-aware commenting

## Requirements

- Neovim 0.8+
- `lunar-lsp` binary (included with Lunar compiler)
- (Optional) [nvim-lspconfig](https://github.com/neovim/nvim-lspconfig) for easier setup

## Installation

### 1. Build the LSP Server

First, build the `lunar-lsp` binary:

```bash
cd /path/to/lunar
go build -o lunar-lsp ./cmd/lunar-lsp

# Move to a directory in your PATH
mv lunar-lsp ~/.local/bin/
# or
sudo mv lunar-lsp /usr/local/bin/
```

### 2. Install the Plugin

#### Using lazy.nvim

```lua
{
  dir = "/path/to/lunar/editors/nvim",
  ft = "lunar",
  config = function()
    require("lunar").setup({
      -- Optional: specify path to lunar-lsp if not in PATH
      -- lsp_binary = "/path/to/lunar-lsp",
    })
  end,
}
```

#### Using packer.nvim

```lua
use {
  "/path/to/lunar/editors/nvim",
  ft = "lunar",
  config = function()
    require("lunar").setup()
  end,
}
```

#### Using vim-plug

```vim
Plug '/path/to/lunar/editors/nvim'

" In your init.lua or after/plugin:
lua require("lunar").setup()
```

#### Manual Installation

Copy or symlink the plugin to your Neovim config:

```bash
# Option 1: Symlink
ln -s /path/to/lunar/editors/nvim ~/.config/nvim/pack/lunar/start/lunar

# Option 2: Copy
cp -r /path/to/lunar/editors/nvim ~/.config/nvim/pack/lunar/start/lunar
```

Then add to your `init.lua`:

```lua
require("lunar").setup()
```

## Configuration

```lua
require("lunar").setup({
  -- Path to lunar-lsp binary (searches PATH if nil)
  lsp_binary = nil,

  -- Enable/disable features
  diagnostics = true,
  hover = true,
  completion = true,
  definition = true,

  -- Auto-attach to lunar files
  auto_attach = true,
})
```

## Key Mappings

The plugin sets up these buffer-local mappings when attached to a Lunar file:

| Key | Action |
|-----|--------|
| `K` | Show hover information |
| `gd` | Go to definition |
| `gD` | Go to declaration |
| `gr` | Find references |
| `gi` | Go to implementation |
| `<leader>rn` | Rename symbol |
| `<leader>ca` | Code actions |
| `[d` | Previous diagnostic |
| `]d` | Next diagnostic |
| `<leader>e` | Show diagnostic float |

## Commands

- `:LunarSetup` - Manually trigger LSP setup
- `:LunarInfo` - Show LSP connection status

## Usage with nvim-cmp

If you use [nvim-cmp](https://github.com/hrsh7th/nvim-cmp) for completion, add the LSP source:

```lua
require("cmp").setup({
  sources = {
    { name = "nvim_lsp" },
    -- ... other sources
  },
})
```

## Troubleshooting

### LSP not starting

1. Check if `lunar-lsp` is in your PATH:
   ```bash
   which lunar-lsp
   ```

2. Check Neovim LSP logs:
   ```vim
   :LspLog
   ```

3. Verify the plugin is loaded:
   ```vim
   :LunarInfo
   ```

### No diagnostics appearing

1. Check if the file has a `.lunar` extension
2. Check `:LspInfo` to see if the client is attached
3. Try saving the file to trigger diagnostics

## License

Same license as the Lunar compiler.
