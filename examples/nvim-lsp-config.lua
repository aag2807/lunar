-- Example Neovim configuration for Lunar LSP
-- Place this in ~/.config/nvim/init.lua or in a separate file in ~/.config/nvim/lua/

-- ============================================================================
-- Option 1: Minimal configuration (no plugins)
-- ============================================================================

-- Auto-detect .lunar files and start LSP
vim.api.nvim_create_autocmd({"BufRead", "BufNewFile"}, {
  pattern = "*.lunar",
  callback = function()
    vim.bo.filetype = "lunar"

    -- Start LSP client
    local client_id = vim.lsp.start({
      name = 'lunar-lsp',
      cmd = {'/home/user/lunar/lunar-lsp'},  -- Adjust path to your lunar-lsp binary
      root_dir = vim.fn.getcwd(),
      capabilities = vim.lsp.protocol.make_client_capabilities(),
    })

    if client_id then
      -- Set up keybindings
      local bufnr = vim.api.nvim_get_current_buf()
      local opts = { noremap=true, silent=true, buffer=bufnr }

      -- LSP keybindings
      vim.keymap.set('n', 'gd', vim.lsp.buf.definition, opts)
      vim.keymap.set('n', 'K', vim.lsp.buf.hover, opts)
      vim.keymap.set('n', 'gr', vim.lsp.buf.references, opts)
      vim.keymap.set('n', '<leader>rn', vim.lsp.buf.rename, opts)
      vim.keymap.set('n', '<leader>ca', vim.lsp.buf.code_action, opts)
      vim.keymap.set('i', '<C-Space>', vim.lsp.buf.completion, opts)

      -- Diagnostics
      vim.keymap.set('n', '[d', vim.diagnostic.goto_prev, opts)
      vim.keymap.set('n', ']d', vim.diagnostic.goto_next, opts)
      vim.keymap.set('n', '<leader>e', vim.diagnostic.open_float, opts)
      vim.keymap.set('n', '<leader>q', vim.diagnostic.setloclist, opts)

      print("Lunar LSP started!")
    end
  end,
})

-- ============================================================================
-- Option 2: Using nvim-lspconfig (recommended)
-- ============================================================================

-- First install nvim-lspconfig:
-- Using lazy.nvim:
--   { 'neovim/nvim-lspconfig' }
-- Using packer:
--   use 'neovim/nvim-lspconfig'

local lspconfig = require('lspconfig')
local configs = require('lspconfig.configs')

-- Define Lunar LSP configuration
if not configs.lunar_lsp then
  configs.lunar_lsp = {
    default_config = {
      cmd = {'/home/user/lunar/lunar-lsp'},  -- Path to lunar-lsp binary
      filetypes = {'lunar'},
      root_dir = lspconfig.util.root_pattern('.git', 'lunar.config.json', '.lunarrc'),
      settings = {
        lunar = {
          -- Custom settings (if your LSP supports them)
          diagnostics = {
            enable = true,
          },
          completion = {
            enable = true,
          },
        },
      },
    },
  }
end

-- Set up Lunar LSP
lspconfig.lunar_lsp.setup({
  on_attach = function(client, bufnr)
    -- Enable completion triggered by <c-x><c-o>
    vim.api.nvim_buf_set_option(bufnr, 'omnifunc', 'v:lua.vim.lsp.omnifunc')

    -- Keybindings
    local opts = { noremap=true, silent=true, buffer=bufnr }

    -- Navigation
    vim.keymap.set('n', 'gD', vim.lsp.buf.declaration, opts)
    vim.keymap.set('n', 'gd', vim.lsp.buf.definition, opts)
    vim.keymap.set('n', 'K', vim.lsp.buf.hover, opts)
    vim.keymap.set('n', 'gi', vim.lsp.buf.implementation, opts)
    vim.keymap.set('n', '<C-k>', vim.lsp.buf.signature_help, opts)

    -- Workspace
    vim.keymap.set('n', '<leader>wa', vim.lsp.buf.add_workspace_folder, opts)
    vim.keymap.set('n', '<leader>wr', vim.lsp.buf.remove_workspace_folder, opts)
    vim.keymap.set('n', '<leader>wl', function()
      print(vim.inspect(vim.lsp.buf.list_workspace_folders()))
    end, opts)

    -- Type definition and references
    vim.keymap.set('n', '<leader>D', vim.lsp.buf.type_definition, opts)
    vim.keymap.set('n', 'gr', vim.lsp.buf.references, opts)

    -- Refactoring
    vim.keymap.set('n', '<leader>rn', vim.lsp.buf.rename, opts)
    vim.keymap.set('n', '<leader>ca', vim.lsp.buf.code_action, opts)

    -- Formatting
    vim.keymap.set('n', '<leader>f', function()
      vim.lsp.buf.format { async = true }
    end, opts)

    -- Diagnostics
    vim.keymap.set('n', '[d', vim.diagnostic.goto_prev, opts)
    vim.keymap.set('n', ']d', vim.diagnostic.goto_next, opts)
    vim.keymap.set('n', '<leader>e', vim.diagnostic.open_float, opts)
    vim.keymap.set('n', '<leader>q', vim.diagnostic.setloclist, opts)

    print("Lunar LSP attached to buffer " .. bufnr)
  end,

  capabilities = vim.lsp.protocol.make_client_capabilities(),

  -- Custom handlers (optional)
  handlers = {
    -- Customize hover window
    ["textDocument/hover"] = vim.lsp.with(
      vim.lsp.handlers.hover,
      { border = "rounded" }
    ),

    -- Customize signature help
    ["textDocument/signatureHelp"] = vim.lsp.with(
      vim.lsp.handlers.signature_help,
      { border = "rounded" }
    ),
  },
})

-- ============================================================================
-- Option 3: Full configuration with nvim-cmp for autocompletion
-- ============================================================================

-- Required plugins:
-- { 'neovim/nvim-lspconfig' }
-- { 'hrsh7th/nvim-cmp' }
-- { 'hrsh7th/cmp-nvim-lsp' }
-- { 'hrsh7th/cmp-buffer' }
-- { 'hrsh7th/cmp-path' }
-- { 'L3MON4D3/LuaSnip' }

local cmp = require('cmp')
local cmp_lsp = require('cmp_nvim_lsp')

-- Configure nvim-cmp
cmp.setup({
  snippet = {
    expand = function(args)
      require('luasnip').lsp_expand(args.body)
    end,
  },
  mapping = cmp.mapping.preset.insert({
    ['<C-b>'] = cmp.mapping.scroll_docs(-4),
    ['<C-f>'] = cmp.mapping.scroll_docs(4),
    ['<C-Space>'] = cmp.mapping.complete(),
    ['<C-e>'] = cmp.mapping.abort(),
    ['<CR>'] = cmp.mapping.confirm({ select = true }),
    ['<Tab>'] = cmp.mapping.select_next_item(),
    ['<S-Tab>'] = cmp.mapping.select_prev_item(),
  }),
  sources = cmp.config.sources({
    { name = 'nvim_lsp' },
    { name = 'luasnip' },
  }, {
    { name = 'buffer' },
    { name = 'path' },
  })
})

-- Enhanced capabilities with nvim-cmp
local capabilities = cmp_lsp.default_capabilities()

-- Set up Lunar LSP with enhanced capabilities
lspconfig.lunar_lsp.setup({
  capabilities = capabilities,
  on_attach = function(client, bufnr)
    -- Same keybindings as Option 2...
    print("Lunar LSP with enhanced completion attached!")
  end,
})

-- ============================================================================
-- Additional configurations
-- ============================================================================

-- Auto-format on save (optional)
vim.api.nvim_create_autocmd("BufWritePre", {
  pattern = "*.lunar",
  callback = function()
    vim.lsp.buf.format({ async = false })
  end,
})

-- Show diagnostics in floating window on cursor hold
vim.api.nvim_create_autocmd("CursorHold", {
  pattern = "*.lunar",
  callback = function()
    vim.diagnostic.open_float(nil, { focusable = false })
  end,
})

-- Configure diagnostic display
vim.diagnostic.config({
  virtual_text = true,
  signs = true,
  underline = true,
  update_in_insert = false,
  severity_sort = true,
  float = {
    border = "rounded",
    source = "always",
    header = "",
    prefix = "",
  },
})

-- Define diagnostic signs
local signs = { Error = "󰅚 ", Warn = "󰀪 ", Hint = "󰌶 ", Info = " " }
for type, icon in pairs(signs) do
  local hl = "DiagnosticSign" .. type
  vim.fn.sign_define(hl, { text = icon, texthl = hl, numhl = hl })
end

-- ============================================================================
-- Utility functions
-- ============================================================================

-- Toggle diagnostics
vim.api.nvim_create_user_command('LunarToggleDiagnostics', function()
  vim.diagnostic.enable(not vim.diagnostic.is_enabled())
end, {})

-- Restart LSP
vim.api.nvim_create_user_command('LunarRestartLsp', function()
  vim.cmd('LspRestart lunar-lsp')
end, {})

-- Show LSP info
vim.api.nvim_create_user_command('LunarLspInfo', function()
  vim.cmd('LspInfo')
end, {})

print("Lunar LSP configuration loaded!")
