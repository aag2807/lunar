-- Library results that are nil only on failure, used as Lua code uses
-- them: directly, or checked with assert or `or`.
print(string.len(math.type(1)))
local path = os.getenv("PATH")
print(#(path .. "/x") > 2)
local name = os.tmpname()
local f = io.open(name, "w")
f:write("x")
f:close()
local g = assert(io.open(name))
print(g:read("a"))
g:close()
os.remove(name)
print(string.upper(io.type(io.stdout)))
print(math.tointeger(3) + 1)
print(type(os.setlocale()))
local home = os.getenv("LUNAR_SURELY_UNSET") or "default"
print(home)
