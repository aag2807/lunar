-- A string method called with '.' and the string as its first argument is
-- plain Lua (string.sub(s, 2, 3)): it stays as written.
local s = "hello"
print(s.sub(s, 2, 3), s.len(s), s.upper(s), s.rep(s, 2, "-"))
local t = { name = "ada" }
print(t.name.upper(t.name))
