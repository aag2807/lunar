-- Lua's operator precedence and associativity, level by level.
local n = 5
print("left " .. n - 1)
print(-2 ^ 2, 2 ^ 3 ^ 2, (2 ^ 3) ^ 2, 2 * 3 ^ 2, 2 ^ -1)
print(not nil == true, not (1 == 2), not 1 == 2)
print("a" .. "b" .. "c", 1 + 2 .. "x", "x" .. 2 * 3)
print(1 | 2 & 3, (1 | 2) & 3, 1 << 2 + 1, 5 // 2 * 2, 7 - 2 - 1, 7 - (2 - 1))
print(1 < 2 == true, 2 ~= 3 and 4 or 5, nil or false and 1)
print(3 ~ 5 | 1, ~0 & 0xF, -(2 + 3), (1 + 2) * 3, 2 ^ (1 + 1))
print(8 // (2 * 2), 8 % (3 + 2), (7 - 2) - 1, 2 ^ (3 ^ 2) == 2 ^ 3 ^ 2)
local a, b, c = "p", "q", "pq"
print((a .. b) == c, a .. b == c, (a or b) and c, a or b and c)
print(("a" .. "b"):upper(), #"abc" + 1, -#"ab", - -3)
print(1 .. 2 == "12", 2 < 3 and 3 < 4, (1 << 4) >> 2, 1 << (4 >> 2))
