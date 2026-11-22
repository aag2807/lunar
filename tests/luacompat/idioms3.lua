-- More everyday Lua: file handles, nil for an optional argument, labels,
-- (...), comparisons in argument lists, callable tables, 'as' as a name.
local path = os.tmpname()
local f = assert(io.open(path, "w"))
f:setvbuf("full", 1024)
f:write("one\n", "two\n")
print(f:seek("cur"), f:seek("set", 0))
f:close()
local r = assert(io.open(path, "r"))
print(r:read("l"), r:read("a"))
r:close()
for line in io.lines(path) do io.write("[", line, "]") end
print()
local h = assert(io.open(path))
for line in h:lines("L") do io.write(#line, " ") end
print()
h:close()
os.remove(path)
io.stdout:write("stdout ok\n")
print(io.stdout:seek("cur") ~= false)

print(load("return 7", nil, "t", {})())
print(select("#", os.setlocale(nil, "numeric")))

do
    local x
    ::top::
    x = (x or 0) + 1
    if x < 3 then goto top end
    print("label", x)
end

local function first(...) return (...) end
local function count(...) local a = (...) return a, select("#", ...) end
print(first(4, 5), count(6, 7))

local a, b, c, d = 1, 2, 3, 4
print(a < b, c > d)
local lt, gt = a < b, c > d
print(lt, gt)

local Stack = { size = 0 }
setmetatable(Stack, { __call = function(self, n) return n * 2 end,
                      __add = function(x, y) return 10 end,
                      __unm = function() return -1 end })
print(Stack(5), Stack + 1, -Stack)

local as = 5;
as = as + 1
print(as)
