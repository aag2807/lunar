-- Every standard library function, in the forms the manual gives, used the
-- way Lua code uses them (Lua 5.3 and 5.4).

-- basic
print(type(print), tostring(12), tonumber("0x10"), tonumber("10", 2), tonumber("z", 36), tonumber("x"))
print(select("#", 1, 2, 3), select(2, "a", "b", "c"), select(-1, "a", "b"))
print(rawequal("a", "a"), rawlen({ 1, 2 }), rawlen("abc"), rawget({ x = 1 }, "x"))
local raw = rawset({}, "k", 5)
print(raw.k, next({}), type(next({ 1 })))
local ok, err = pcall(error, "boom", 0)
print(ok, err)
print(xpcall(function() error({ code = 7 }) end, function(e) return e.code end))
print(pcall(error))
print(assert(1, "unused"), select("#", assert(true, 2, 3)))
local mt = { __index = function(_, k) return k .. "!" end }
local obj = setmetatable({}, mt)
print(obj.hi, getmetatable(obj) == mt, getmetatable("").__index == string)
local chunk = load("return 1 + ...")
print(chunk(41), load("syntax error here"))
local env = { x = 3 }
print(load("return x", "chunk", "t", env)())
local pieces = { "return ", "7" }
print(load(function() return table.remove(pieces, 1) end)())
print(type(collectgarbage("count")), collectgarbage("collect"), collectgarbage("step"))
for i, v in ipairs({ "a", "b" }) do io.write(i, v, " ") end
local keys = {}
for k in pairs({ x = 1, y = 2 }) do keys[#keys + 1] = k end
table.sort(keys)
print(table.concat(keys, ","))
print(_VERSION:sub(1, 4), type(_G), _G.print == print)

-- string
local s = "Hello, Lua"
print(string.len(s), s:len(), #s, s:upper(), s:lower(), s:reverse())
print(s:sub(1, 5), s:sub(-3), s:sub(8), ("x"):rep(3), ("ab"):rep(3, "-"))
print(s:byte(1), s:byte(1, 3), string.char(72, 105))
print(s:find("Lua"), s:find("l+"), s:find(".", 1, true), s:find("zz"))
print(s:match("(%a+), (%a+)"), s:match("%d"), ("key=val"):match("(%w+)=(%w+)"))
print(string.format("%d %5.2f %s %q %x %%", 42, 3.14159, "str", "a\nb", 255))
print(string.format("%-5s|%05d|%+d|%e", "ab", 42, 7, 12345.678))
print(s:gsub("l", "L"), s:gsub("l", "L", 1), ("abc"):gsub("%w", "%0%0"))
print(("hello world"):gsub("%w+", { hello = "HI" }), ("abc"):gsub(".", function(c) return c:byte() end))
for word in ("one two three"):gmatch("%a+") do io.write(word, ";") end
for k, v in ("a=1, b=2"):gmatch("(%w+)=(%w+)") do io.write(k, v, ";") end
print()
local packed = string.pack("i4", 100)
print(#packed, string.unpack("i4", packed), string.packsize("i4"))
print(type(string.dump(function() return 1 end)))

-- table
local list = { 3, 1, 2 }
table.insert(list, 4)
table.insert(list, 1, 0)
print(table.concat(list, " "), table.remove(list), table.remove(list, 1), #list)
table.sort(list)
print(table.concat(list, ","))
table.sort(list, function(a, b) return a > b end)
print(table.concat(list, ","), table.concat({ 1, 2, 3 }, "-", 2, 3))
print(table.unpack({ 1, 2, 3 }), table.unpack({ 1, 2, 3 }, 2), select("#", table.unpack({ 1, nil, 3 }, 1, 3)))
local packedt = table.pack(1, nil, 3)
print(packedt.n, packedt[3])
print(table.concat(table.move({ 1, 2, 3 }, 1, 3, 2, {}), ",", 2, 4))

-- math
print(math.floor(3.7), math.ceil(3.2), math.abs(-4), math.max(1, 5, 3), math.min(4, 2))
print(math.sqrt(16), math.huge > 0, -math.huge < 0, math.pi > 3, math.maxinteger > 0, math.mininteger < 0)
print(math.fmod(7, 3), math.modf(3.5), math.tointeger(3.0), math.tointeger(3.5), math.type(1), math.type(1.0), math.type("1"))
print(math.ult(1, 2), math.exp(0), math.log(1), math.log(8, 2), math.log(100, 10))
print(math.sin(0), math.cos(0), math.tan(0), math.asin(0), math.acos(1), math.atan(0), math.atan(1, 1) > 0)
print(math.deg(math.pi), math.rad(180) == math.pi)
math.randomseed(42)
local r1 = math.random()
local r2 = math.random(10)
local r3 = math.random(5, 6)
print(r1 >= 0 and r1 < 1, r2 >= 1 and r2 <= 10, r3 == 5 or r3 == 6)

-- utf8
local u = "héllo"
print(utf8.len(u), utf8.char(72, 233), utf8.codepoint(u, 2), utf8.offset(u, 3), #utf8.charpattern > 0)
for p, c in utf8.codes("aé") do io.write(p, ":", c, " ") end
print()

-- os
print(type(os.time()), type(os.clock()), type(os.date()), os.date("%Y", 0) == "1970" or os.date("%Y", 0) == "1969")
print(os.time({ year = 2020, month = 1, day = 1, hour = 12 }) > 0, os.date("*t", 0).year >= 1969, os.date("!%H", 0))
print(os.difftime(10, 5), type(os.getenv("PATH")), os.getenv("LUNAR_NOT_SET") == nil)
local tmp = os.tmpname()
print(type(tmp), os.remove(tmp) or true, select("#", os.remove(tmp)))
print(type(os.setlocale()), os.setlocale(nil, "numeric") ~= nil, type(os.execute()))

-- io
local name = os.tmpname()
local out = assert(io.open(name, "w"))
print(io.type(out), io.type(42))
out:write("line1\n", 2, "\n", "line3")
out:setvbuf("full", 1024)
out:flush()
out:close()
print(io.type(out))
local inp = assert(io.open(name, "r"))
print(inp:read("l"), inp:read("n"), inp:read("L"), inp:read("a"), inp:read("a"), inp:read("l"))
print(inp:seek("set", 0), inp:seek("cur"), inp:seek("end"))
inp:seek("set")
for line in inp:lines() do io.write("[", line, "]") end
print()
inp:close()
for a, b in io.lines(name, 1, 2) do io.write(a, b, "|") break end
print()
for line in io.lines(name, "L") do io.write(#line, " ") end
print()
io.output():write("")
print(io.output() == io.stdout, io.input() == io.stdin)
local tf = io.tmpfile()
tf:write("tmp")
tf:seek("set")
print(tf:read("a"))
tf:close()
os.remove(name)
io.write("io.write ", 1, "\n")
io.stdout:write("stdout ", 2, "\n")

-- coroutine
local co = coroutine.create(function(a, b)
    local c = coroutine.yield(a + b)
    return c * 2
end)
print(coroutine.resume(co, 1, 2))
print(coroutine.status(co), coroutine.resume(co, 10), coroutine.status(co))
print(coroutine.resume(co))
local gen = coroutine.wrap(function() for i = 1, 3 do coroutine.yield(i) end end)
print(gen(), gen(), gen())
print(coroutine.isyieldable(), type(coroutine.running()), select(2, coroutine.running()))

-- debug
local info = debug.getinfo(1, "S")
print(info.what, type(debug.getinfo(print).what), type(debug.traceback()))
print(debug.getinfo(1, "l") ~= nil, type(debug.getlocal(1, 1)))
local function up() return info end
print(debug.getupvalue(up, 1), type(debug.getregistry()), debug.getmetatable("").__index == string)
print(debug.gethook())
debug.sethook()
debug.setmetatable(10, nil)
print(type(debug.setupvalue(up, 1, info)))

-- package
print(type(package.path), type(package.cpath), type(package.loaded), package.loaded.string == string)
print(type(package.preload), type(package.searchers), #package.config > 0, type(package.searchpath))
print(require("string") == string)
