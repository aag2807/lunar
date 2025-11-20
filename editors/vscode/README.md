# Lunar Language Support for VS Code

Provides language support for Lunar - a statically-typed superset of Lua.

## Features

- Syntax highlighting
- Real-time error diagnostics
- Go to definition
- Hover type information
- Auto-completion

## Requirements

1. Build the Lunar LSP server:
   ```bash
   cd /path/to/lunar
   go build -o lunar-lsp ./cmd/lunar-lsp
   ```

2. Add `lunar-lsp` to your PATH or configure the path in settings.

## Installation

### From Source

1. Clone the repository
2. Install dependencies:
   ```bash
   cd editors/vscode
   npm install
   ```
3. Copy to VS Code extensions folder:
   ```bash
   cp -r . ~/.vscode/extensions/lunar-lang
   ```
4. Restart VS Code

### From VSIX (coming soon)

```bash
code --install-extension lunar-lang-1.5.0.vsix
```

## Configuration

| Setting | Description | Default |
|---------|-------------|---------|
| `lunar.serverPath` | Path to lunar-lsp executable | `lunar-lsp` |
| `lunar.trace.server` | Enable LSP tracing | `off` |

## Commands

- `Lunar: Restart Server` - Restart the language server

## Development

```bash
# Install dependencies
npm install

# Package extension
npx vsce package
```

## License

MIT
