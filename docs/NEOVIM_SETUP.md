# Lunar LSP + Neovim Integration Guide

Complete guide for setting up Lunar language server with Neovim for a full IDE experience.

## Prerequisites

- **Neovim >= 0.8.0** (LSP support is built-in)
- **Lunar compiler installed** (in your PATH)
- **Lunar LSP server** (will be built)

## Installation

### 1. Build Lunar LSP Server

```bash
cd /path/to/lunar
go build -o lunar-lsp ./cmd/lunar-lsp
sudo mv lunar-lsp /usr/local/bin/
```

Verify installation:
```bash
lunar-lsp --version
```

### 2. Configure Neovim

Neovim has built-in LSP support. You just need to configure it to use the Lunar LSP server.

#### Option A: Using Lazy.nvim (Recommended)

Create or edit `~/.config/nvim/lua/plugins/lunar.lua`:

```lua
return {
  -- LSP configuration
  {
    "neovim/nvim-lspconfig",
    dependencies = {
      -- Autocompletion
      "hrsh7th/nvim-cmp",
      "hrsh7th/cmp-nvim-lsp",
      "hrsh7th/cmp-buffer",
      "hrsh7th/cmp-path",
      "L3MON4D3/LuaSnip",
    },
    config = function()
      local lspconfig = require("lspconfig")
      local configs = require("lspconfig.configs")

      -- Define Lunar LSP
      if not configs.lunar then
        configs.lunar = {
          default_config = {
            cmd = { "lunar-lsp" },
            filetypes = { "lunar" },
            root_dir = function(fname)
              return lspconfig.util.find_git_ancestor(fname) or vim.fn.getcwd()
            end,
            settings = {
              lunar = {
                diagnostics = {
                  enable = true,
                  debounce = 300,
                },
                completion = {
                  enable = true,
                },
              },
            },
          },
        }
      end

      -- Setup Lunar LSP
      lspconfig.lunar.setup({
        on_attach = function(client, bufnr)
          -- Enable completion triggered by <c-x><c-o>
          vim.api.nvim_buf_set_option(bufnr, "omnifunc", "v:lua.vim.lsp.omnifunc")

          -- Keybindings
          local opts = { noremap = true, silent = true, buffer = bufnr }

          -- Go to definition
          vim.keymap.set("n", "gd", vim.lsp.buf.definition, opts)

          -- Go to type definition
          vim.keymap.set("n", "gD", vim.lsp.buf.type_definition, opts)

          -- Hover documentation
          vim.keymap.set("n", "K", vim.lsp.buf.hover, opts)

          -- Go to implementation
          vim.keymap.set("n", "gi", vim.lsp.buf.implementation, opts)

          -- Signature help
          vim.keymap.set("n", "<C-k>", vim.lsp.buf.signature_help, opts)

          -- Rename symbol
          vim.keymap.set("n", "<leader>rn", vim.lsp.buf.rename, opts)

          -- Code action
          vim.keymap.set("n", "<leader>ca", vim.lsp.buf.code_action, opts)

          -- Find references
          vim.keymap.set("n", "gr", vim.lsp.buf.references, opts)

          -- Format
          vim.keymap.set("n", "<leader>f", function()
            vim.lsp.buf.format({ async = true })
          end, opts)

          -- Diagnostics
          vim.keymap.set("n", "<leader>e", vim.diagnostic.open_float, opts)
          vim.keymap.set("n", "[d", vim.diagnostic.goto_prev, opts)
          vim.keymap.set("n", "]d", vim.diagnostic.goto_next, opts)
        end,
        capabilities = require("cmp_nvim_lsp").default_capabilities(),
      })

      -- Autocompletion setup
      local cmp = require("cmp")
      cmp.setup({
        snippet = {
          expand = function(args)
            require("luasnip").lsp_expand(args.body)
          end,
        },
        mapping = cmp.mapping.preset.insert({
          ["<C-b>"] = cmp.mapping.scroll_docs(-4),
          ["<C-f>"] = cmp.mapping.scroll_docs(4),
          ["<C-Space>"] = cmp.mapping.complete(),
          ["<C-e>"] = cmp.mapping.abort(),
          ["<CR>"] = cmp.mapping.confirm({ select = true }),
        }),
        sources = cmp.config.sources({
          { name = "nvim_lsp" },
          { name = "luasnip" },
        }, {
          { name = "buffer" },
          { name = "path" },
        }),
      })

      -- Diagnostic configuration
      vim.diagnostic.config({
        virtual_text = true,
        signs = true,
        update_in_insert = false,
        underline = true,
        severity_sort = true,
        float = {
          border = "rounded",
          source = "always",
          header = "",
          prefix = "",
        },
      })

      -- Diagnostic signs
      local signs = { Error = "󰅚 ", Warn = "󰀪 ", Hint = "󰌶 ", Info = " " }
      for type, icon in pairs(signs) do
        local hl = "DiagnosticSign" .. type
        vim.fn.sign_define(hl, { text = icon, texthl = hl, numhl = hl })
      end
    end,
  },

  -- Optional: Treesitter for better syntax highlighting
  {
    "nvim-treesitter/nvim-treesitter",
    build = ":TSUpdate",
    config = function()
      -- We'll need a Lunar parser for Treesitter
      -- For now, use Lua parser as fallback
      vim.treesitter.language.register("lua", "lunar")
    end,
  },

  -- Optional: File type detection
  {
    "nathom/filetype.nvim",
    config = function()
      require("filetype").setup({
        overrides = {
          extensions = {
            lunar = "lunar",
          },
        },
      })
    end,
  },
}
```

#### Option B: Using Packer.nvim

Add to `~/.config/nvim/lua/plugins.lua`:

```lua
use {
  "neovim/nvim-lspconfig",
  requires = {
    "hrsh7th/nvim-cmp",
    "hrsh7th/cmp-nvim-lsp",
  },
  config = function()
    -- Same configuration as above
  end
}
```

#### Option C: Manual Configuration (No Plugin Manager)

Create `~/.config/nvim/lua/lunar-lsp.lua`:

```lua
-- Same LSP configuration code as above
```

Then add to your `init.lua`:
```lua
require("lunar-lsp")
```

### 3. File Type Detection

Create `~/.config/nvim/ftdetect/lunar.vim`:

```vim
au BufRead,BufNewFile *.lunar set filetype=lunar
```

Or in Lua, add to `~/.config/nvim/init.lua`:

```lua
vim.filetype.add({
  extension = {
    lunar = "lunar",
  },
})
```

### 4. Syntax Highlighting (Basic)

Until we have a Treesitter parser, create basic syntax highlighting:

Create `~/.config/nvim/syntax/lunar.vim`:

```vim
" Lunar syntax highlighting
if exists("b:current_syntax")
  finish
endif

" Keywords
syn keyword lunarKeyword class interface enum type function const local return if else elseif then end for while do
syn keyword lunarKeyword repeat until break declare implements extends private public constructor self
syn keyword lunarType string number boolean nil any void table thread

" Types
syn match lunarType "\<[A-Z][a-zA-Z0-9]*\>"

" Strings
syn region lunarString start=+"+ skip=+\\\\\|\\"+ end=+"+
syn region lunarString start=+'+ skip=+\\\\\|\\'+ end=+'+

" Numbers
syn match lunarNumber "\<\d\+\>"
syn match lunarNumber "\<\d\+\.\d*\>"
syn match lunarNumber "\<\.\d\+\>"

" Comments
syn match lunarComment "--.*$"
syn region lunarComment start="--\[\[" end="\]\]"

" Operators
syn match lunarOperator "[+\-*/%^#=<>~]"
syn match lunarOperator "\.\."

" Functions
syn match lunarFunction "\<[a-z_][a-zA-Z0-9_]*\s*("he=e-1

" Highlight groups
hi def link lunarKeyword Keyword
hi def link lunarType Type
hi def link lunarString String
hi def link lunarNumber Number
hi def link lunarComment Comment
hi def link lunarOperator Operator
hi def link lunarFunction Function

let b:current_syntax = "lunar"
```

## Keybindings Reference

Once configured, you'll have these keybindings in Lunar files:

| Key | Action |
|-----|--------|
| `gd` | Go to definition |
| `gD` | Go to type definition |
| `K` | Show hover documentation |
| `gi` | Go to implementation |
| `<C-k>` | Signature help |
| `<leader>rn` | Rename symbol |
| `<leader>ca` | Code action |
| `gr` | Find references |
| `<leader>f` | Format document |
| `<leader>e` | Show diagnostic float |
| `[d` | Go to previous diagnostic |
| `]d` | Go to next diagnostic |
| `<C-Space>` | Trigger completion |

(Note: `<leader>` is typically `\` or `<Space>` depending on your config)

## Usage Examples

### 1. Open a Lunar File

```bash
nvim examples/todo_app.lunar
```

The LSP will automatically start and provide diagnostics.

### 2. Get Type Information

Place cursor on a variable and press `K` to see its type.

### 3. Go to Definition

Place cursor on a function call and press `gd` to jump to its definition.

### 4. Autocomplete

Type `string.` and press `<C-Space>` to see string library methods.

### 5. Find References

Place cursor on a function name and press `gr` to see all usages.

### 6. Rename Symbol

Place cursor on a variable and press `<leader>rn` to rename it everywhere.

## Advanced Configuration

### Custom LSP Settings

```lua
lspconfig.lunar.setup({
  settings = {
    lunar = {
      diagnostics = {
        enable = true,
        debounce = 500,  -- Wait 500ms before checking
      },
      completion = {
        enable = true,
        snippets = true,
      },
      trace = {
        server = "verbose",  -- For debugging
      },
    },
  },
})
```

### Statusline Integration

With `lualine.nvim`:

```lua
require("lualine").setup({
  sections = {
    lualine_c = {
      "filename",
      {
        "diagnostics",
        sources = { "nvim_lsp" },
        symbols = { error = "E:", warn = "W:", info = "I:", hint = "H:" },
      },
    },
  },
})
```

### Telescope Integration

For better reference/symbol search:

```lua
-- Find workspace symbols
vim.keymap.set("n", "<leader>ws", function()
  require("telescope.builtin").lsp_workspace_symbols()
end)

-- Find document symbols
vim.keymap.set("n", "<leader>ds", function()
  require("telescope.builtin").lsp_document_symbols()
end)
```

## Troubleshooting

### LSP Not Starting

Check if server is running:
```vim
:LspInfo
```

Enable LSP logs:
```lua
vim.lsp.set_log_level("debug")
```

View logs:
```bash
tail -f ~/.local/state/nvim/lsp.log
```

### Autocompletion Not Working

Verify nvim-cmp is installed:
```vim
:checkhealth nvim-cmp
```

### No Diagnostics

Check if diagnostics are enabled:
```vim
:lua =vim.diagnostic.is_disabled()
```

### Server Crashes

Check server logs:
```bash
lunar-lsp --stdio --verbose
```

## Project Structure

For best LSP experience, structure your project like this:

```
my-lunar-project/
├── .git/               # LSP uses this as project root
├── lua.d.lunar        # Copy from stdlib/
├── src/
│   ├── main.lunar
│   ├── utils.lunar
│   └── types.d.lunar  # Type declarations
├── test/
│   └── test.lunar
└── build/
    └── *.lua          # Compiled output
```

## Example Workflow

1. **Open project**: `nvim src/main.lunar`
2. **LSP starts automatically** and shows any errors
3. **Start coding** with autocompletion
4. **See errors inline** with red squiggly lines
5. **Hover** to see type information
6. **Jump to definitions** with `gd`
7. **Refactor** with rename and code actions
8. **Compile** with `:!lunar % -o build/`
9. **Run** with `:!lua build/%.lua`

## Additional Plugins

### Recommended

- **nvim-lspconfig** - LSP configuration (required)
- **nvim-cmp** - Autocompletion (required)
- **telescope.nvim** - Fuzzy finder for symbols/references
- **trouble.nvim** - Pretty diagnostics list
- **lspsaga.nvim** - Enhanced LSP UI
- **null-ls.nvim** - Formatting/linting integration

### Nice to Have

- **nvim-treesitter** - Better syntax highlighting (needs Lunar parser)
- **lualine.nvim** - Statusline with LSP info
- **gitsigns.nvim** - Git integration
- **which-key.nvim** - Keybinding help

## Minimal Configuration

If you want the absolute minimum setup:

```lua
-- ~/.config/nvim/init.lua
local lspconfig = require("lspconfig")
local configs = require("lspconfig.configs")

-- Define Lunar LSP
configs.lunar = {
  default_config = {
    cmd = { "lunar-lsp" },
    filetypes = { "lunar" },
    root_dir = lspconfig.util.find_git_ancestor,
  },
}

-- Start LSP
lspconfig.lunar.setup({})

-- Keybindings
vim.keymap.set("n", "gd", vim.lsp.buf.definition)
vim.keymap.set("n", "K", vim.lsp.buf.hover)

-- File type
vim.filetype.add({ extension = { lunar = "lunar" } })
```

That's it! You now have basic LSP support.

## Next Steps

1. Build the LSP server (once implemented)
2. Configure Neovim with one of the options above
3. Start coding in Lunar with IDE features!
4. Report issues/improvements

## Resources

- [Neovim LSP Documentation](https://neovim.io/doc/user/lsp.html)
- [LSP Specification](https://microsoft.github.io/language-server-protocol/)
- [nvim-lspconfig](https://github.com/neovim/nvim-lspconfig)
- [nvim-cmp](https://github.com/hrsh7th/nvim-cmp)
