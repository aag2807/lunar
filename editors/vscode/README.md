# Lunar for VS Code

Syntax highlighting (TextMate grammar) and the Lunar language server
(`lunar lsp`): diagnostics, go to definition, references, rename, hover,
completion, signature help and symbols.

## Build and install

The [2.1.2 release](https://github.com/aag2807/lunar/releases/tag/v2.1.2)
has the packaged extension, `lunar-2.1.2.vsix`. To build it yourself:

```sh
cd editors/vscode
npm ci
npm run package        # bundles with esbuild and writes lunar-2.1.2.vsix
```

Then in VS Code: *Extensions* → `…` → *Install from VSIX…*.

### Project in WSL, VS Code on Windows

Open the folder through the WSL extension (`code .` from a WSL shell). The
extension declares `"extensionKind": ["workspace"]`, so it installs and runs
inside WSL and starts the Linux `lunar`; install the VSIX from that WSL
window. `lunar` must be on the WSL `PATH` (the Linux release archive, or
`make install PREFIX=$HOME/.local`), or set `lunar.serverPath`.

## Settings

- `lunar.serverPath` (default `lunar`): the executable; the server runs as
  `<serverPath> lsp`.
- `lunar.trace.server`: `off`, `messages` or `verbose`.

Command: **Lunar: Restart Language Server**.

## Build task errors

The `lunar` problem matcher reads `lunar build --json`, so a task like this
lists build errors in the Problems panel:

```json
{
  "label": "lunar build",
  "type": "shell",
  "command": "lunar build --json",
  "problemMatcher": "$lunar"
}
```

The matcher reads the JSON lines with a pattern, so a message keeps the
JSON escapes of any `"` in it, and columns count bytes (as `--json`
documents): on a line with non-ASCII text the underline may sit a little to
the right. The language server's own diagnostics have neither problem.
