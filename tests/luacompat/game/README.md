# The game's scripts

The 14 Lua scripts the engine ships, unchanged, with the declarations of the
engine globals they use (`api/`) and the engine side's recording harness
(`harness.lua`). `TestGameScriptsBehaveTheSame` compiles them as `.lunar`
files with no diagnostics allowed, then runs each original and its compiled
Lua under the harness, which records every call to an engine global, and
requires the two records to be identical.

Under the harness `level2` and `level3` never finish, in either form (their
loops wait on values the recorder does not give), so for them the test
requires that neither form finishes within the time limit.
