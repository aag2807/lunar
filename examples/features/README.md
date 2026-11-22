# Feature examples

One program per language feature, written to be run rather than read. Each
ends with asserts on the values it computes, so a wrong answer fails as well as
a crash. `tests/examples` compiles every one for `lua53` and `lua54` and runs it
under Lua 5.3 and 5.4; `go test ./tests/examples` does the same locally.

```sh
for f in examples/features/*.lunar; do
    lunar "$f" && (cd examples/features && lua5.4 "$(basename "${f%.lunar}").lua")
done
```

`mod/math_utils.lunar` is imported by `17_modules.lunar`; compile it too.
