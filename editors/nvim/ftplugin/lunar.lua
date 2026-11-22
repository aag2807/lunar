-- Settings for Lunar buffers.
vim.bo.commentstring = '-- %s'
vim.bo.comments = ':---,:--'

-- Tree-sitter highlighting and folding, when the parser is installed
-- (:TSInstall lunar). Without it the regex syntax in syntax/lunar.lua is used.
if pcall(vim.treesitter.start) then
  vim.wo[0][0].foldexpr = 'v:lua.vim.treesitter.foldexpr()'
  if pcall(require, 'nvim-treesitter') then
    vim.bo.indentexpr = "v:lua.require'nvim-treesitter'.indentexpr()"
  end
end

vim.b.undo_ftplugin = 'setlocal commentstring< comments< indentexpr<'
