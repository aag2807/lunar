-- The Lunar language server, for vim.lsp.config / vim.lsp.enable (Neovim 0.11+).
-- Override with vim.lsp.config('lunar', { cmd = { '/path/to/lunar', 'lsp' } }).
return {
  cmd = { 'lunar', 'lsp' },
  filetypes = { 'lunar' },
  root_markers = { 'lunar.config.json', '.git' },
}
