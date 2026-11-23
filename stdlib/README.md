# Standard library declarations

These files describe Lua 5.3 and 5.4's standard library to the type checker.
They are compiled into the `lunar` binary (`embed.go`), so they are always
available and nothing needs to be installed next to it.

| File | |
|---|---|
| `lua.d.lunar` | the basic functions: `print`, `type`, `pairs`, `load`, `pcall`, ... |
| `string.d.lunar`, `table.d.lunar`, `math.d.lunar`, `utf8.d.lunar` | those libraries |
| `os.d.lunar`, `io.d.lunar`, `coroutine.d.lunar`, `debug.d.lunar`, `package.d.lunar` | those libraries |
| `lua54.d.lunar` | what only Lua 5.4 has (`warn`); loaded for `--target lua54` |
| `lua53.d.lunar` | what only Lua 5.3 has (`bit32`, from 5.2); loaded for `--target lua53` |

A function that returns `nil` (and a message) only to report failure, such
as `io.open`, `os.getenv` or `math.type`, is declared with what it returns on
success, as Teal does. Code checks the failure the way Lua code does
(`assert(io.open(p))`, `os.getenv(k) or default`), and plain Lua that uses
the result directly compiles. An engine's own declarations can say `T?`
where they want callers to check.

The list of files loaded is `Core` and `ForTarget` in `embed.go`. Errors in
them are reported as `<stdlib>/<file>`, and a test checks they load with none.

Declarations for anything else, such as an engine's API or a Lua library,
belong in the project: any `*.d.lunar` under `rootDir`, or a folder named in
`lunar.config.json`'s `declarations`. See
[`docs/ENGINE_INTEGRATION.md`](../docs/ENGINE_INTEGRATION.md).
