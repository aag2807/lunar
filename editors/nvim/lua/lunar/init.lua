local M = {}

-- The tree-sitter grammar's folder, if this plugin lives in a checkout of
-- the Lunar repository (editors/nvim next to editors/tree-sitter-lunar).
function M.local_grammar()
  local here = debug.getinfo(1, 'S').source:sub(2)
  local plugin_root = vim.fs.dirname(vim.fs.dirname(vim.fs.dirname(here)))
  local grammar = vim.fs.joinpath(vim.fs.dirname(plugin_root), 'tree-sitter-lunar')
  if vim.uv.fs_stat(vim.fs.joinpath(grammar, 'src', 'parser.c')) then
    return grammar
  end
end

-- install_info for nvim-treesitter's parser table.
function M.parser_install_info()
  local grammar = M.local_grammar()
  if grammar then
    return { path = grammar, queries = 'queries' }
  end
  return {
    url = 'https://github.com/aag2807/lunar',
    location = 'editors/tree-sitter-lunar',
    queries = 'queries',
  }
end

-- Optional configuration. `cmd` is the lunar binary to run; the server is
-- started with `<cmd> lsp`.
function M.setup(opts)
  opts = opts or {}
  if opts.cmd then
    vim.lsp.config('lunar', { cmd = { opts.cmd, 'lsp' } })
  end
end

return M
