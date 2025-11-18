-- Lunar LSP Configuration for Neovim

local M = {}

-- Find the lunar-lsp binary
local function find_lsp_binary(config_path)
  if config_path and vim.fn.executable(config_path) == 1 then
    return config_path
  end

  -- Check if lunar-lsp is in PATH
  if vim.fn.executable("lunar-lsp") == 1 then
    return "lunar-lsp"
  end

  -- Check common installation locations
  local locations = {
    vim.fn.expand("~/.local/bin/lunar-lsp"),
    vim.fn.expand("~/go/bin/lunar-lsp"),
    "/usr/local/bin/lunar-lsp",
    "/usr/bin/lunar-lsp",
  }

  for _, loc in ipairs(locations) do
    if vim.fn.executable(loc) == 1 then
      return loc
    end
  end

  return nil
end

function M.setup(config)
  local lsp_binary = find_lsp_binary(config.lsp_binary)

  if not lsp_binary then
    vim.notify(
      "lunar-lsp not found. Please install it or set lsp_binary in config.",
      vim.log.levels.WARN
    )
    return
  end

  -- Register lunar-lsp with nvim-lspconfig if available
  local has_lspconfig, lspconfig = pcall(require, "lspconfig")
  local has_configs, configs = pcall(require, "lspconfig.configs")

  if has_lspconfig and has_configs then
    -- Register custom LSP server
    if not configs.lunar_lsp then
      configs.lunar_lsp = {
        default_config = {
          cmd = { lsp_binary },
          filetypes = { "lunar" },
          root_dir = function(fname)
            return lspconfig.util.root_pattern(".git", "lunar.json", "*.lunar")(fname)
              or vim.fn.getcwd()
          end,
          settings = {},
        },
      }
    end

    -- Setup the LSP server
    lspconfig.lunar_lsp.setup({
      on_attach = function(client, bufnr)
        -- Enable completion triggered by <c-x><c-o>
        vim.bo[bufnr].omnifunc = "v:lua.vim.lsp.omnifunc"

        -- Buffer local mappings
        local opts = { buffer = bufnr, noremap = true, silent = true }

        if config.hover then
          vim.keymap.set("n", "K", vim.lsp.buf.hover, opts)
        end

        if config.definition then
          vim.keymap.set("n", "gd", vim.lsp.buf.definition, opts)
          vim.keymap.set("n", "gD", vim.lsp.buf.declaration, opts)
        end

        if config.completion then
          -- Completion is handled by omnifunc or completion plugins
        end

        -- Additional useful mappings
        vim.keymap.set("n", "gr", vim.lsp.buf.references, opts)
        vim.keymap.set("n", "gi", vim.lsp.buf.implementation, opts)
        vim.keymap.set("n", "<leader>rn", vim.lsp.buf.rename, opts)
        vim.keymap.set("n", "<leader>ca", vim.lsp.buf.code_action, opts)
        vim.keymap.set("n", "[d", vim.diagnostic.goto_prev, opts)
        vim.keymap.set("n", "]d", vim.diagnostic.goto_next, opts)
        vim.keymap.set("n", "<leader>e", vim.diagnostic.open_float, opts)

        vim.notify("Lunar LSP attached to buffer " .. bufnr, vim.log.levels.INFO)
      end,

      capabilities = vim.lsp.protocol.make_client_capabilities(),
    })
  else
    -- Fallback: use vim.lsp.start directly without lspconfig
    vim.api.nvim_create_autocmd("FileType", {
      pattern = "lunar",
      callback = function(args)
        if not config.auto_attach then
          return
        end

        vim.lsp.start({
          name = "lunar-lsp",
          cmd = { lsp_binary },
          root_dir = vim.fn.getcwd(),
          on_attach = function(client, bufnr)
            vim.bo[bufnr].omnifunc = "v:lua.vim.lsp.omnifunc"

            local opts = { buffer = bufnr, noremap = true, silent = true }
            vim.keymap.set("n", "K", vim.lsp.buf.hover, opts)
            vim.keymap.set("n", "gd", vim.lsp.buf.definition, opts)
            vim.keymap.set("n", "[d", vim.diagnostic.goto_prev, opts)
            vim.keymap.set("n", "]d", vim.diagnostic.goto_next, opts)
          end,
        })
      end,
    })
  end

  -- Configure diagnostics display
  if config.diagnostics then
    vim.diagnostic.config({
      virtual_text = true,
      signs = true,
      underline = true,
      update_in_insert = false,
      severity_sort = true,
    })

    -- Define diagnostic signs
    local signs = {
      Error = "E",
      Warn = "W",
      Hint = "H",
      Info = "I",
    }

    for type, icon in pairs(signs) do
      local hl = "DiagnosticSign" .. type
      vim.fn.sign_define(hl, { text = icon, texthl = hl, numhl = hl })
    end
  end
end

return M
