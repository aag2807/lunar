-- needs: lua54
-- What only Lua 5.4's standard library has.
warn("@on")
warn("@off")
local co = coroutine.create(function() coroutine.yield(1) end)
coroutine.resume(co)
print(coroutine.close(co), coroutine.status(co))
print(utf8.len("\xff", 1, -1, true), utf8.len("abc", 1, -1, false))
for p, c in utf8.codes("ab", true) do io.write(p, c, " ") end
print()
print(utf8.codepoint("é", 1, -1, true))
print(select("#", math.randomseed(7)), os.difftime(10, 4))
do
    local closed = {}
    do
        local x <close> = setmetatable({}, { __close = function() closed[#closed + 1] = "x" end })
        local y <const> = 2
        print("inside", y)
    end
    print(closed[1])
end
print(math.type(3 // 1), 7 // 2, 7.0 // 2, 2^2, 5 & 3, 5 | 3, 5 ~ 3, ~0, 1 << 4, 256 >> 4)
print(string.format("%a", 1) ~= nil, ("%5.1s|"):format("abc"))
print(type(debug.setcstacklimit), #string.pack("j", 1))
