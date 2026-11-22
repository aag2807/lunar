# Lunar for Neovim

Language server (diagnostics, go to definition, references, rename, hover,
completion, signature help, symbols) through Neovim's built-in LSP client,
and highlighting, folding and indentation through tree-sitter.

Needs Neovim 0.11 or newer, and nvim-treesitter's `main` branch for the parser.

## Install

1. Put `lunar` on your `PATH`; the plugin runs `lunar lsp`.

   ```sh
   make install PREFIX=$HOME/.local   # in the Lunar repository
   ```

2. Add the plugin. With lazy.nvim (for NvChad, in `lua/plugins/init.lua`):

   ```lua
   {
     dir = "~/personal/lunar/editors/nvim", -- this folder, in your checkout
     lazy = false, -- it is small, and must register the parser before :TSInstall
     dependencies = { "nvim-treesitter/nvim-treesitter" },
   },
   ```

   From a checkout, the parser is built from the repository's own grammar
   (`editors/tree-sitter-lunar`); otherwise it is fetched from GitHub.

3. Build the parser, then restart Neovim:

   ```vim
   :TSInstall lunar
   ```

4. Check everything is found:

   ```vim
   :checkhealth lunar
   ```

## Configure

The language server is enabled for `lunar` buffers with `vim.lsp.enable('lunar')`
when the plugin loads. Its configuration is `lsp/lunar.lua` and can be extended
like any other:

```lua
vim.lsp.config("lunar", {
  cmd = { "/path/to/lunar", "lsp" }, -- if lunar is not on PATH
})
```

Set `vim.g.lunar_lsp = false` before the plugin loads to enable it yourself
(or not at all). The project root is the nearest folder with a
`lunar.config.json`, or `.git`.

Without the tree-sitter parser, buffers fall back to the regex syntax in
`syntax/lunar.lua`.

Folding: the plugin sets `foldexpr` to tree-sitter's folds, and leaves
`foldmethod` to you (setting it for you would open every buffer with all its
folds closed). To fold Lunar buffers by syntax:

```lua
vim.api.nvim_create_autocmd("FileType", {
  pattern = "lunar",
  callback = function() vim.wo[0][0].foldmethod = "expr" end,
})
```

## Test

```sh
editors/nvim/test/run.sh   # headless: builds lunar and the parser, then checks
                           # highlighting, attach, definition and diagnostics
```
