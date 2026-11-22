-- needs: lua54
-- ';' between fields, goto and labels, and Lua 5.4's attributes.
local t = { [1] = "a"; "b"; c = 3, }
print(t[1], t.c)
for i = 1, 3 do
    if i == 2 then goto continue end
    print(i)
    ::continue::
end
do
    local k = 0
    ::top::
    k = k + 1
    if k < 3 then goto top end
    print(k)
end
local x <const> = 5
print(x)
local closed = false
do
    local h <close> = setmetatable({}, { __close = function() closed = true end })
end
print(closed)
