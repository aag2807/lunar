-- Everyday untyped Lua: defaults with or, cond and a or b, locals filled
-- later, tables grown field by field, varargs, closures, generators.
local c = {}
c.attempt = (c.attempt or 0) + 1
c.attempt = (c.attempt or 0) + 1
local name = c.name or "anonymous"
local label = c.attempt > 1 and "retry" or "first"
print(c.attempt, name, label, nil or 5, false or nil, 1 and 2, nil and 1)

local x
x = 5
local best = nil
for _, v in ipairs({ 3, 9, 4 }) do
    if not best or v > best then best = v end
end
print(x + 1, best)

local t = { a = 1 }
t.b = 2
t.name = "x"
print(t.a + t.b, t.name, t.missing)

local list = {}
list[#list + 1] = 3
table.insert(list, 4)
table.insert(list, 1, 2)
print(#list, list[1] + list[2] + list[3], table.remove(list), #list)

local function f(a, b)
    b = b or 2
    return a * b
end
print(f(3), f(3, 3))

local function sum(...)
    local n = 0
    for i = 1, select("#", ...) do n = n + select(i, ...) end
    return n, select("#", ...)
end
print(sum(1, 2, 3))
local packed = table.pack(1, 2, 3)
print(packed.n, table.unpack(packed, 1, packed.n))
local a, b, extra = 1
print(a, b, extra)
local first = sum(4, 5)
print(first)

local parts = {}
for w in ("a,b,c"):gmatch("[^,]+") do parts[#parts + 1] = w end
print(table.concat(parts, "-"), (("hello"):gsub("l", function(ch) return ch:upper() end)))
print(string.format("%5.2f|%d|%s", 3.14159, 42, "x"), ("abc"):sub(-2), ("A"):byte(), string.char(66))

local ok, err = pcall(error, { code = 7 })
print(ok, err.code)
local ok2, msg = pcall(function() error("boom", 0) end)
print(ok2, msg)

local memo = {}
local function fib(n)
    if n < 2 then return n end
    if memo[n] then return memo[n] end
    local v = fib(n - 1) + fib(n - 2)
    memo[n] = v
    return v
end
print(fib(30))

local function counter()
    local n = 0
    return function() n = n + 1; return n end
end
local nxt = counter()
nxt()
print(nxt())

local gen = coroutine.wrap(function() for i = 1, 3 do coroutine.yield(i) end end)
print(gen(), gen(), gen())

local M = {}
function M.add(p, q) return p + q end
function M.twice(p) return M.add(p, p) end
print(M.twice(4))

local handlers = { add = function(p, q) return p + q end }
handlers.sub = function(p, q) return p - q end
print(handlers.add(1, 2), handlers["sub"](5, 3))

local keys = {}
for k in pairs({ b = 1, a = 2, c = 3 }) do keys[#keys + 1] = k end
table.sort(keys)
table.sort(list, function(p, q) return p > q end)
print(table.concat(keys, ","), list[1])

local i = 0
repeat
    local done = i >= 3
    i = i + 1
until done
print(i, tostring(nil) .. tostring(12), (tonumber("42") or 0) + 1, math.floor(7 / 2), 7 // 2, 7 % 3)

local q = {}
q.first, q.last = 1, 0
q.last = q.last + 1
q[q.last] = "job"
print(q[q.first], #q)
