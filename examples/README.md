# Examples

| Folder / file | |
|---|---|
| [`features/`](features/) | One program per language feature. Each asserts its own results; CI compiles them for `lua53` and `lua54` and runs them under Lua 5.3 and 5.4. |
| [`data/`](data/) | A data file, loaded by CI in a Lua state with no libraries opened, as an engine loads data. |
| `todo_app.lunar` | A small program: an enum, an interface, a class, optional parameters, a tuple return, pattern matching. Run in CI too. |
| `love_bouncing_balls.lunar`, `love_game_example.lunar` | LÖVE games, typed against `love.d.lunar`. Run them with `love`. |
| the other `*.lunar` files | Tours of single features. They compile in CI; their generated Lua is pinned in `tests/examples/testdata/golden`. |

```sh
lunar features/01_primitives.lunar && lua5.4 features/01_primitives.lua
```

`features/17_modules.lunar` imports `features/mod/math_utils.lunar`; compile
both, and run from `features/` so `require("mod.math_utils")` finds it.
