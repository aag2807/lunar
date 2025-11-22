-- Lunar Language LSP Configuration for Neovim
-- Add this to your Neovim configuration (init.lua or init.vim)

-- For init.lua users:
-- You can source this file directly: dofile('path/to/lunar/nvim-lsp-config.lua')
-- Or copy the relevant sections to your config

local function setup_lunar_lsp()
  -- Get the absolute path to the lunar-lsp binary
  -- Adjust this path based on where you installed lunar-lsp
  local lunar_lsp_cmd = 'lunar-lsp'

  -- You can also use an absolute path:
  -- local lunar_lsp_cmd = '/usr/local/bin/lunar-lsp'
  -- or for the local build:
  -- local lunar_lsp_cmd = vim.fn.getcwd() .. '/lunar-lsp'

  -- Check if lspconfig is installed
  local ok, lspconfig = pcall(require, 'lspconfig')
  if not ok then
    vim.notify('nvim-lspconfig is not installed. Please install it first.', vim.log.levels.ERROR)
    return
  end

  -- Check if lspconfig.configs is available
  if not lspconfig.configs then
    lspconfig.configs = {}
  end

  -- Define the Lunar LSP configuration
  if not lspconfig.configs.lunar then
    lspconfig.configs.lunar = {
      default_config = {
        cmd = { lunar_lsp_cmd },
        filetypes = { 'lunar' },
        root_dir = function(fname)
          return lspconfig.util.find_git_ancestor(fname) or vim.fn.getcwd()
        end,
        settings = {},
        single_file_support = true,
      },
    }
  end

  -- Setup the Lunar LSP
  lspconfig.lunar.setup({
    on_attach = function(client, bufnr)
      -- Enable completion triggered by <c-x><c-o>
      vim.api.nvim_buf_set_option(bufnr, 'omnifunc', 'v:lua.vim.lsp.omnifunc')

      -- Mappings
      local bufopts = { noremap=true, silent=true, buffer=bufnr }
      vim.keymap.set('n', 'gD', vim.lsp.buf.declaration, bufopts)
      vim.keymap.set('n', 'gd', vim.lsp.buf.definition, bufopts)
      vim.keymap.set('n', 'K', vim.lsp.buf.hover, bufopts)
      vim.keymap.set('n', 'gi', vim.lsp.buf.implementation, bufopts)
      vim.keymap.set('n', '<C-k>', vim.lsp.buf.signature_help, bufopts)
      vim.keymap.set('n', '<space>wa', vim.lsp.buf.add_workspace_folder, bufopts)
      vim.keymap.set('n', '<space>wr', vim.lsp.buf.remove_workspace_folder, bufopts)
      vim.keymap.set('n', '<space>wl', function()
        print(vim.inspect(vim.lsp.buf.list_workspace_folders()))
      end, bufopts)
      vim.keymap.set('n', '<space>D', vim.lsp.buf.type_definition, bufopts)
      vim.keymap.set('n', '<space>rn', vim.lsp.buf.rename, bufopts)
      vim.keymap.set('n', '<space>ca', vim.lsp.buf.code_action, bufopts)
      vim.keymap.set('n', 'gr', vim.lsp.buf.references, bufopts)
      vim.keymap.set('n', '<space>f', function() vim.lsp.buf.format { async = true } end, bufopts)

      print('Lunar LSP attached to buffer ' .. bufnr)
    end,
    capabilities = require('cmp_nvim_lsp').default_capabilities(vim.lsp.protocol.make_client_capabilities()),
    flags = {
      debounce_text_changes = 150,
    }
  })

  -- Set up filetype detection for .lunar files
  vim.filetype.add({
    extension = {
      lunar = 'lunar',
    },
  })

  print('Lunar LSP configuration loaded')
end

-- Call the setup function
setup_lunar_lsp()

-- For vim users (init.vim):
-- You'll need to use:
-- lua dofile('path/to/lunar/nvim-lsp-config.lua')
