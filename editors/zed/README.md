# Lunar for Zed

Highlighting, brackets, indentation and an outline from the tree-sitter grammar
in `editors/tree-sitter-lunar`, and the Lunar language server (`lunar lsp`).

## Install as a dev extension

1. Put `lunar` on your `PATH`: from a release archive, or
   `make install PREFIX=$HOME/.local` in the repository. On Windows with the project in WSL, open the folder in Zed's
   WSL mode so the server runs inside WSL.
2. Install Rust through rustup and add the WebAssembly target:
   `rustup target add wasm32-wasip2`. Zed builds the extension with the Rust
   on its own side: for Zed on Windows, that is rustup installed on Windows,
   not the one inside WSL.
3. In Zed: *Extensions* → *Install Dev Extension* → choose `editors/zed`.

To check the extension compiles without Zed:
`cargo build --release --target wasm32-wasip2` in `editors/zed`.

`extension.toml` fetches the grammar from GitHub at `rev`. When the grammar
changes, set `rev` to a pushed commit that has the change.

`languages/lunar/highlights.scm` is a copy of the grammar's
`queries/highlights.scm`; CI fails if they differ, so edit the grammar's and
copy it here.
