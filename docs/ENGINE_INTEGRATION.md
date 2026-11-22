# Running Lunar from a game engine

This is the contract between Lunar and a program that runs it: an engine's
editor, a development build, a build pipeline. The program starts `lunar` as
a child process and reads its output; nothing links against the compiler, and
a shipped game never needs it.

## The project

A project is a folder of scripts with a `lunar.config.json`. Paths in it are
relative to the file.

```json
{
  "rootDir": "scripts",
  "outDir": "../build/scripts",
  "target": "lua54",
  "declarations": ["../engine/api"],
  "copyLua": true,
  "sourceMap": false,
  "dataFiles": ["data/**"]
}
```

| Option | Meaning | Default |
|---|---|---|
| `rootDir` | The scripts folder. Imports and `require` names are relative to it. | the config's folder |
| `outDir` | Where the Lua goes, mirroring `rootDir`'s layout. Required for `lunar build`, and not the same folder as `rootDir`. | none |
| `target` | `lua53` or `lua54`. | `lua53` |
| `declarations` | Extra `*.d.lunar` files or folders, anywhere on disk: the engine's API. | none |
| `copyLua` | Copy plain `.lua` scripts under `rootDir` to `outDir`. | `true` |
| `sourceMap` | Write `<file>.lua.map` next to each output. | `false` |
| `dataFiles` | Globs (relative to `rootDir`, `**` crosses folders) of scripts loaded with no libraries opened. | none |

A key that is not in this table is an error, so a misspelt option fails
instead of being ignored.

## Building

```sh
lunar build --json
```

`lunar` finds `lunar.config.json` by walking up from the working directory;
`--project path/to/lunar.config.json` names it instead. One run:

1. reads every `*.lunar` file under `rootDir` and every declaration file;
2. checks them all, following imports, so an error that crosses files is
   caught;
3. **only if there are no errors**, writes every output: each file is written
   next to its target and renamed over it, so nothing ever reads half a file.

A failed build writes nothing. The Lua from the last good build stays in
place, so a running game keeps working on it while the script is fixed. That
holds when writing itself fails, too: if one file cannot be replaced (on
Windows, a file another program has open; renames are retried briefly
first), every file already replaced is put back and the run exits 3 with
"... (nothing was changed)".

The output folder mirrors `rootDir`. Plain `.lua` scripts are copied through
(a `foo.lua` next to a `foo.lunar` is an error, since both would write
`foo.lua`). `outDir/.lunar-manifest.json` records what the build wrote: when a
script is deleted, the next build removes its output, and nothing lunar did not
write is ever touched. Declaration files produce no output.

Flags:

| Flag | |
|---|---|
| `--json` | Machine-readable output (below). |
| `--watch` | Rebuild whenever a script, plain Lua file or declaration changes. |
| `--out <dir>` | Override `outDir`. |
| `--target lua53\|lua54` | Override `target`. |
| `--decl <path>` | Add a declaration file or folder; repeatable. |
| `--source-map` | Write source maps. |

## Output

With `--json`, stdout carries one JSON object per line and stderr stays empty.
Each problem is a `diagnostic` line, and every build ends with one `summary`
line (in watch mode, every rebuild does):

```
{"kind":"diagnostic","file":"/game/scripts/enemy.lunar","line":5,"column":37,"endLine":5,"endColumn":42,"severity":"error","code":"field-type","message":"Field 'hp' of 'Enemy' must be 'number', got 'string'"}
{"kind":"summary","errors":1,"warnings":0,"files":14,"written":false}
```

- The fields always come in this order, so a simple reader can rely on it.
- `file` is an absolute path. Errors in the bundled standard library read
  `<stdlib>/lua.d.lunar`. A problem that is not about a file (bad usage, an
  I/O failure) has an empty `file` and zero positions.
- `line` and `column` are 1-based; `column` counts bytes. `endLine` and
  `endColumn` are just past the span.
- `severity` is `error`, `warning` or `info`.
- `code` names the kind of problem and does not change between releases.
- `summary.files` is how many scripts were compiled; `written` says whether
  the output folder was updated.

| Code | |
|---|---|
| `syntax` | The script does not parse. |
| `type` | A type error. |
| `field-type` | A table literal's field has the wrong type. |
| `unknown-field` | A table literal has a field its type does not. |
| `missing-fields` | A table literal leaves out required fields. |
| `unknown-member` | A class or declared namespace has no such member. |
| `duplicate-function` | A second definition of a function. |
| `import` | An import names no module, or something the module does not export. |
| `import-cycle` | Imports form a cycle, which `require` cannot load. |
| `module-name` | A file name `require` cannot express (a `.` in a folder or file name). |
| `output-conflict` | A `.lua` script and a `.lunar` one would write the same file. |
| `data-file` | A data file uses something a state with no libraries lacks. |
| `not-supported` | `async`/`await`. |
| `usage` | Bad flags or arguments. |
| `config` | A bad or missing `lunar.config.json`, or no `outDir`. |
| `io` | Reading or writing failed. |
| `internal` | A compiler bug. |

Without `--json`, errors go to stderr with a line of context around each.

### Exit codes

| Code | |
|---|---|
| `0` | Compiled; the output is up to date. |
| `1` | The scripts have errors; nothing was written. |
| `2` | Usage or configuration error. |
| `3` | I/O failure or internal error. |

### Version

`lunar --version` prints one line, `lunar 2.0.0` followed by the git revision
it was built from when known: `lunar 2.0.0 (1a2b3c4d5e6f)`. An editor can
compare the version with the one it expects.

## Loading the Lua

`require` names are relative to `rootDir`: `scripts/ai/brain.lunar` is
`require("ai.brain")`, and `scripts/ai/init.lunar` is `require("ai")`. Point
`package.path` at the output folder:

```lua
package.path = "build/scripts/?.lua;build/scripts/?/init.lua"
```

An exported function, class or constant is a field of the table the module
returns, never a global.

## Data files

A data file is a script whose chunk returns a table:

```lunar
interface EnemyDef
    name: string
    hp: number
    speed: number?
end

local goblin: EnemyDef = {
    name = "goblin",
    hp = 10,
}

return goblin
```

The engine can load it with `luaL_newstate()` and no `luaL_openlibs`. Files
matching `dataFiles` are checked for that: a use of any global (`tostring`,
the engine's API) or of a construct whose Lua needs one (a class needs
`setmetatable`, a spread `table.unpack`, a template string `tostring`, `match`
`error`), or a string method (`name:upper()`, which needs the string
library's metatable), is a `data-file` error. Interfaces cost nothing at run
time.

## Declaring the engine's API

Declarations describe Lua the engine provides. They are signatures only:

```lunar
declare function log(message: string): void

declare namespace engine
    function spawn(kind: string, x?: number, y?: number): number
    function despawn(id: number): void
    const frame: number
end

declare class Sprite
    x: number
    y: number
    constructor(path: string)
    draw(): void
    static load(path: string): Sprite
end
```

- A trailing `end` after `declare function` is accepted but not needed.
- Members of a declared namespace are reached through it (`engine.spawn`), and
  one it does not list is an error.
- A declared class describes a Lua table the engine made, so calling it is
  emitted as written (`Sprite("hero.png")` stays `Sprite("hero.png")`);
  declare `static new(...)` if the engine's constructor is `Sprite.new`.
- Overloads are written as several `declare function` signatures with the
  same name.

## Calling methods

The checker decides whether a call needs Lua's `:` (an instance method) or `.`
(a function in a table). When a value's type is `any`, it cannot know, and the
call is emitted as written: write `obj:method()` to pass `self`.

- A function in a table literal is a plain field: `M.greet("bob")`.
- A method declared in an interface or class takes `self`; a table literal
  that implements one writes `function(self, ...)`.
- A union calls through `:` or `.` when all its types agree, and is an error
  when they do not.
- A method has to be called where it is named: `local f = e.hit` is an
  error, since `f(3)` would pass 3 as `self`. Wrap it in a function instead.

## Editors

`lunar lsp` is the language server; it reads the same configuration and
declarations as `lunar build`, so the editor and the build agree. Each open
file uses the `lunar.config.json` found from its own folder, so the editor
may open a folder above the game's. A config error is shown on the config
file and as a message. Rename is refused while any file has a syntax error.
See `editors/` for Neovim, VS Code and Zed.

Compiling one file (`lunar file.lunar [-o out.lua]`, flags before or after
the name) uses the config found above it; with none, it sees the
declaration files in its own folder and any given with `--decl`. `-o` may
name a device or pipe, such as `/dev/stdout`.
