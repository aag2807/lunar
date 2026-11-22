-- More Lua the compiler changed or refused (found by fuzzing).
local s = "abc"
print(s.upper("xyz"), s.len("four"))
local function first(...) return (...) end
print(first(1, 2))
local function xor(...) return ... ~ 1 end
print(xor(6))
local function dec(...) return ... - 1 end
print(dec(6))
local n = 0
local function f()
  ::retry::
  n = n + 1
  if n < 3 then goto retry end
  return n
end
print(f())
local a, b = 1, 2
print(a < b >> 1)
local function assert(v) return v end
local function print2(...) print(...) end
print2(assert(1))
local p = { 1, tonumber("2") }
print(p[1] + p[2])
do
  function inblock() return "global from a block" end
end
print(inblock())
local function g(x) if x then return g end return 1 end
print(g(nil) + 1, g(true) == g)
local flag = math.type(1) == "integer"
local r29 = { class = (flag and 16 or 7), name = (flag and "a" or "b") }
r29.class = 19
r29.name = "c"
print(r29.class, r29.name)
