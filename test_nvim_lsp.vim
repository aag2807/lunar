" Vim script to test Lunar LSP attachment

" Wait a bit for LSP to attach
sleep 2

" Check if LSP is attached
lua << LUA
local clients = vim.lsp.get_active_clients()
if #clients > 0 then
  for _, client in ipairs(clients) do
    if client.name == 'lunar' then
      print("✓ Lunar LSP is attached!")
      print("  Client ID: " .. client.id)
      print("  Root dir: " .. (client.config.root_dir or "N/A"))
      vim.cmd('qall!')
      return
    end
  end
  print("✗ LSP attached but not Lunar LSP")
  print("  Found: " .. clients[1].name)
else
  print("✗ No LSP clients attached")
end
LUA

" If we get here, something went wrong
qall!
