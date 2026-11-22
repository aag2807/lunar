-- Headless smoke test: nvim --headless --clean with this plugin, a built
-- parser and `lunar` on PATH (see run.sh). Exits non-zero on failure.
local failures = {}
local function check(ok, msg)
  if not ok then
    table.insert(failures, msg)
  end
end

local project = assert(vim.env.LUNAR_TEST_PROJECT)
vim.cmd.edit(project .. '/main.lunar')
local buf = vim.api.nvim_get_current_buf()

check(vim.bo[buf].filetype == 'lunar', 'filetype is ' .. vim.bo[buf].filetype)
check(vim.treesitter.highlighter.active[buf] ~= nil, 'tree-sitter highlighting did not start')

-- `local e: Enemy = ...` on line 3: the annotation is a type. Headless,
-- nothing redraws, so parse explicitly before asking.
vim.treesitter.get_parser(buf):parse(true)
local caps = vim.treesitter.get_captures_at_pos(buf, 2, 10)
local names = vim.tbl_map(function(c) return c.capture end, caps)
check(vim.tbl_contains(names, 'type'), 'annotation captures: ' .. table.concat(names, ','))

local attached = vim.wait(10000, function()
  return #vim.lsp.get_clients({ bufnr = buf, name = 'lunar' }) > 0
end)
check(attached, 'the lunar language server did not attach')

if attached then
  local client = vim.lsp.get_clients({ bufnr = buf, name = 'lunar' })[1]
  -- Go to the definition of Enemy in `Enemy(10)`: actors/enemy.lunar.
  local res = client:request_sync('textDocument/definition', {
    textDocument = { uri = vim.uri_from_bufnr(buf) },
    position = { line = 2, character = 18 },
  }, 10000, buf)
  local uri = res and res.result and (res.result.uri or (res.result[1] and res.result[1].uri))
  check(uri and uri:match('actors/enemy%.lunar$'), 'definition went to ' .. vim.inspect(res))

  -- An error typed into the buffer is reported.
  vim.api.nvim_buf_set_lines(buf, -1, -1, false, { 'local s: string = 1' })
  local got = vim.wait(10000, function()
    return #vim.diagnostic.get(buf) > 0
  end)
  check(got, 'no diagnostics after introducing an error')
end

-- Outside a checkout the parser comes from GitHub, from the repository's
-- branch (nvim-treesitter would otherwise ask for a "main" that is not there).
do
  local lunar = require('lunar')
  local real = lunar.local_grammar
  lunar.local_grammar = function() return nil end
  local info = lunar.parser_install_info()
  lunar.local_grammar = real
  check(info.url and info.branch == 'master', 'GitHub parser source: ' .. vim.inspect(info))
end

if #failures > 0 then
  io.stderr:write('FAIL\n  ' .. table.concat(failures, '\n  ') .. '\n')
  vim.cmd('cquit 1')
else
  io.stdout:write('nvim smoke test OK\n')
  vim.cmd('qall!')
end
