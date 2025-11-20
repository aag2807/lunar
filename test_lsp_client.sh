#!/bin/bash
# Simple script to test the Lunar LSP server

echo "Testing Lunar LSP Server..."
echo ""
echo "1. Starting lunar-lsp in background..."

# Start the LSP server
/home/user/lunar/lunar-lsp > /tmp/lsp-server.log 2>&1 &
LSP_PID=$!

sleep 1

# Check if server is running
if ps -p $LSP_PID > /dev/null; then
    echo "✓ LSP server started (PID: $LSP_PID)"
else
    echo "✗ LSP server failed to start"
    exit 1
fi

echo ""
echo "2. Testing initialize request..."

# Send initialize request
cat << 'EOF' | /home/user/lunar/lunar-lsp > /tmp/lsp-response.txt 2>&1 &
Content-Length: 226

{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"processId":null,"rootUri":"file:///home/user/lunar","capabilities":{"textDocument":{"hover":{},"definition":{},"references":{},"rename":{},"completion":{}}}}}
EOF

sleep 1

echo "✓ Initialize request sent"

# Check server log
echo ""
echo "3. Server log (first 20 lines):"
head -20 /tmp/lsp-server.log

# Cleanup
kill $LSP_PID 2>/dev/null

echo ""
echo "Testing complete!"
echo ""
echo "To test interactively with Neovim:"
echo "  cd /home/user/lunar"
echo "  nvim test_lsp.lunar"
echo ""
echo "Available LSP commands in nvim:"
echo "  - gd: Go to definition"
echo "  - K: Hover (show type info)"
echo "  - gr: Find all references"
echo "  - <leader>rn: Rename symbol"
echo "  - :LspInfo to check LSP status"
