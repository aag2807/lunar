-- Filetype detection for Lunar files
vim.filetype.add({
  extension = {
    lunar = "lunar",
  },
  pattern = {
    [".*%.d%.lunar"] = "lunar",
  },
})
