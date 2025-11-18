-- Lunar plugin entry point
-- This file is automatically loaded by Neovim

-- Prevent loading twice
if vim.g.loaded_lunar then
  return
end
vim.g.loaded_lunar = true

-- Create user command to manually setup
vim.api.nvim_create_user_command("LunarSetup", function()
  require("lunar").setup()
end, { desc = "Setup Lunar LSP" })

-- Create command to show LSP info
vim.api.nvim_create_user_command("LunarInfo", function()
  local clients = vim.lsp.get_clients({ name = "lunar_lsp" })
  if #clients > 0 then
    print("Lunar LSP is active")
    for _, client in ipairs(clients) do
      print("  ID: " .. client.id)
      print("  Root: " .. (client.config.root_dir or "N/A"))
    end
  else
    print("Lunar LSP is not active")
  end
end, { desc = "Show Lunar LSP info" })
