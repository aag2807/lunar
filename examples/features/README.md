# Feature examples

One program per documented feature area, written to be run rather than read.
Each file compiles with the checker enabled and executes under Lua 5.4; between
them they cover the language surface listed in the root README.

```sh
make build
for f in examples/features/*.lunar; do
    ./dist/lunar --no-cache "$f" && lua5.4 "${f%.lunar}.lua"
done
```

`mod/math_utils.lunar` is imported by `17_modules.lunar`; compile it first so
`require` can find the generated module.

These doubled as an audit: writing them surfaced eleven defects, from `local
function` not parsing to `super()` never running the parent constructor. Keeping
them in the repository means the same ground is covered every time they are run.
