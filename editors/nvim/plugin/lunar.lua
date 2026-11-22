-- Lunar support for Neovim 0.11+: the language server through vim.lsp, and
-- the tree-sitter parser through nvim-treesitter (main branch).
if vim.g.loaded_lunar then
  return
end
vim.g.loaded_lunar = true

local lunar = require('lunar')

-- nvim-treesitter learns about the parser here; `:TSInstall lunar` builds it.
-- A checkout of the repository builds from its own grammar; otherwise it is
-- fetched from GitHub.
vim.api.nvim_create_autocmd('User', {
  pattern = 'TSUpdate',
  callback = function()
    local ok, parsers = pcall(require, 'nvim-treesitter.parsers')
    if ok then
      parsers.lunar = { install_info = lunar.parser_install_info() }
    end
  end,
})

-- Start the server for Lunar buffers unless the user opted out with
-- vim.g.lunar_lsp = false (to enable it themselves, or not at all).
if vim.g.lunar_lsp ~= false and vim.lsp.enable then
  vim.lsp.enable('lunar')
end
