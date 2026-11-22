# Lunar for Zed

Highlighting, brackets, indentation and an outline from the tree-sitter grammar
in `editors/tree-sitter-lunar`, and the Lunar language server (`lunar lsp`).

## Install as a dev extension

1. Put `lunar` on your `PATH` (`make install PREFIX=$HOME/.local` in the
   repository). On Windows with the project in WSL, open the folder in Zed's
   WSL mode so the server runs inside WSL.
2. Install Rust through rustup (Zed builds the extension with it) and add
   the WebAssembly target: `rustup target add wasm32-wasip2`.
3. In Zed: *Extensions* → *Install Dev Extension* → choose `editors/zed`.

`extension.toml` fetches the grammar from GitHub at `rev`. Until that commit
is pushed, point `repository` at your checkout:
`repository = "file:///home/you/personal/lunar"`.

`languages/lunar/highlights.scm` is a copy of the grammar's
`queries/highlights.scm`; CI fails if they differ, so edit the grammar's and
copy it here.
