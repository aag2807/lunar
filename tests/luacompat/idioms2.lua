-- Everyday Lua the checker refused.
local modname = ...
print(type(modname))

-- Generic for with any iterator: a list after 'in', closures, gmatch.
local t = { a = 1 }
for k, v in next, t do print(k, v) end
for k, v in next, t, nil do print(k, v) end
local function counter(n)
    local i = 0
    return function()
        i = i + 1
        if i <= n then return i end
    end
end
for i in counter(3) do io.write(i, " ") end
print()
for word in string.gmatch("one two", "%a+") do io.write(word, ";") end
for a, b in ("k=v, x=y"):gmatch("(%w+)=(%w+)") do io.write(a, b, " ") end
print()
local function iter(list)
    return function(s, c)
        c = c + 1
        if s[c] then return c, s[c] end
    end, list, 0
end
for i, v in iter({ "p", "q" }) do io.write(i, v, " ") end
print()

-- String and table keys on an empty table.
local seen = {}
local name = "ada"
seen[name] = true
seen["y"] = 2
seen[tostring(3)] = 4
local key = {}
seen[key] = "table key"
print(seen.ada, seen.y, seen["3"], seen[key])

-- Unannotated functions take what Lua passes them.
local function f(a, ...) return a, select("#", ...) end
print(f())
local function g(a) return a end
print(g(1, 2))
local function h(...) return f(...) end
print(h(1, 2, 3))
print(load("return 1", nil, "t", {})())

-- A local declared first and defined later.
local isodd
local function iseven(n) if n == 0 then return true end return isodd(n - 1) end
function isodd(n) if n == 0 then return false end return iseven(n - 1) end
print(iseven(10), isodd(7))

-- A table local given up later.
local cache = {}
cache = nil
print(cache)
local r = {}
r = { err = "x" }
print(r.err)

-- Locals named like library globals.
local function shout(string) string = string .. "!" return string end
print(shout("hey"))
local table2 = table
local table
table = {}
print(type(table2.concat), next(table))
