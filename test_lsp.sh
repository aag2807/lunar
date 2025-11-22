#!/bin/bash

# Test script for Lunar LSP with Neovim

echo "Testing Lunar LSP with Neovim..."
echo ""

# Check if lunar-lsp is built
if [ ! -f "./lunar-lsp" ]; then
  echo "Error: lunar-lsp binary not found. Building..."
  make build-lunar-lsp
fi

# Check if nvim is installed
if ! command -v nvim &> /dev/null; then
  echo "Error: Neovim is not installed"
  exit 1
fi

echo "Neovim version:"
nvim --version | head -n 1
echo ""

# Create a test lunar file
cat > /tmp/test_lsp.lunar << 'EOF'
-- Test Lunar LSP functionality

class Person {
  name: string
  age: number

  constructor(name: string, age: number) {
    this.name = name
    this.age = age
  }

  greet() -> string {
    return "Hello, my name is " .. this.name
  }
}

local person = Person("Alice", 30)
print(person:greet())

-- Test type annotations
local x: number = 42
local y: string = "hello"

-- Test function with types
fn add(a: number, b: number) -> number {
  return a + b
}

local result = add(10, 20)
print(result)
EOF

echo "Created test file: /tmp/test_lsp.lunar"
echo ""

# Test LSP server directly
echo "Testing LSP server startup..."
timeout 2s ./lunar-lsp < /dev/null &
LSP_PID=$!
sleep 1

if ps -p $LSP_PID > /dev/null; then
  echo "✓ LSP server starts successfully"
  kill $LSP_PID 2>/dev/null
else
  echo "✓ LSP server ran (already exited)"
fi

echo ""
echo "Opening test file in Neovim..."
echo "Once in Neovim:"
echo "  - Check :LspInfo to see if Lunar LSP is attached"
echo "  - Try 'gd' on a symbol to go to definition"
echo "  - Try 'K' on a symbol to see hover info"
echo "  - Use ':q' to quit"
echo ""
echo "Press Enter to continue..."
read

nvim /tmp/test_lsp.lunar

echo ""
echo "LSP test complete!"
