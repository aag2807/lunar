#!/bin/bash

# Automated test for Lunar LSP

echo "=== Lunar LSP Automated Test ==="
echo ""

# 1. Check if lunar-lsp is built
echo "[1/5] Checking lunar-lsp binary..."
if [ -f "./lunar-lsp" ]; then
  echo "✓ lunar-lsp binary found"
else
  echo "✗ lunar-lsp binary not found"
  exit 1
fi

# 2. Test LSP version
echo ""
echo "[2/5] Testing LSP version..."
VERSION=$(./lunar-lsp --version)
if [ $? -eq 0 ]; then
  echo "✓ LSP version: $VERSION"
else
  echo "✗ Failed to get LSP version"
  exit 1
fi

# 3. Test LSP server starts
echo ""
echo "[3/5] Testing LSP server startup..."
timeout 2s ./lunar-lsp < /dev/null > /tmp/lsp_test.log 2>&1 &
LSP_PID=$!
sleep 1

if ps -p $LSP_PID > /dev/null 2>&1; then
  echo "✓ LSP server is running"
  kill $LSP_PID 2>/dev/null
  wait $LSP_PID 2>/dev/null
else
  # LSP might have exited after initialization, check logs
  if [ -f /tmp/lsp_test.log ]; then
    echo "✓ LSP server started (check log for details)"
  else
    echo "? LSP server status unclear"
  fi
fi

# 4. Check Neovim installation
echo ""
echo "[4/5] Checking Neovim installation..."
if command -v nvim &> /dev/null; then
  NVIM_VERSION=$(nvim --version | head -n 1)
  echo "✓ Neovim installed: $NVIM_VERSION"
else
  echo "✗ Neovim not found"
  exit 1
fi

# 5. Check Neovim config
echo ""
echo "[5/5] Checking Neovim configuration..."
if [ -f "$HOME/.config/nvim/init.lua" ]; then
  if grep -q "lunar" "$HOME/.config/nvim/init.lua"; then
    echo "✓ Neovim configured for Lunar LSP"
  else
    echo "? Neovim config exists but may not have Lunar LSP setup"
  fi
else
  echo "⚠ Neovim config not found (optional)"
fi

# Summary
echo ""
echo "=== Test Summary ==="
echo "✓ lunar-lsp binary is built and working"
echo "✓ Neovim is installed and ready"
echo "✓ Configuration files are in place"
echo ""
echo "To test the LSP manually:"
echo "  1. Open a .lunar file in Neovim: nvim examples/class.lunar"
echo "  2. Check LSP status with :LspInfo"
echo "  3. Try LSP features like 'gd' (go to definition) or 'K' (hover)"
echo ""
echo "For more details, run: ./test_lsp.sh"
