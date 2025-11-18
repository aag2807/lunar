-- Lunar Neovim Plugin
-- Provides LSP integration for the Lunar programming language

local M = {}

M.config = {
  -- Path to lunar-lsp binary (will search in PATH if not specified)
  lsp_binary = nil,
  -- Enable/disable features
  diagnostics = true,
  hover = true,
  completion = true,
  definition = true,
  -- Auto-attach to lunar files
  auto_attach = true,
}

function M.setup(opts)
  M.config = vim.tbl_deep_extend("force", M.config, opts or {})

  -- Setup LSP
  require("lunar.lsp").setup(M.config)
end

return M
