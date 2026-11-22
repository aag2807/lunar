-- :checkhealth lunar
local M = {}

function M.check()
  local health = vim.health
  health.start('lunar')

  local config = vim.lsp.config.lunar
  local exe = config and config.cmd and config.cmd[1] or 'lunar'
  if vim.fn.executable(exe) == 1 then
    local out = vim.fn.system({ exe, '--version' })
    health.ok(('%s found: %s'):format(exe, vim.trim(out)))
  else
    health.error(('%s is not on PATH'):format(exe), {
      'Build it with `make build` in the Lunar repository and put dist/ on PATH,',
      "or point the plugin at it: vim.lsp.config('lunar', { cmd = { '/path/to/lunar', 'lsp' } })",
    })
  end

  if vim.lsp.is_enabled and vim.lsp.is_enabled('lunar') then
    health.ok('language server enabled (vim.lsp.enable)')
  else
    health.warn("language server not enabled: call vim.lsp.enable('lunar')")
  end

  if pcall(vim.treesitter.language.add, 'lunar') then
    health.ok('tree-sitter parser installed')
  else
    health.warn('tree-sitter parser not installed', { 'Run :TSInstall lunar (nvim-treesitter main branch)' })
  end
end

return M
