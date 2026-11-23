-- needs: lua53
-- What only Lua 5.3's standard library has: bit32, kept from Lua 5.2.
print(bit32.band(12, 10), bit32.bor(12, 10), bit32.bxor(12, 10), bit32.bnot(0))
print(bit32.lshift(1, 4), bit32.rshift(256, 4), bit32.arshift(-16, 2), bit32.btest(4, 6))
print(bit32.extract(0xF0, 4, 4), bit32.replace(0, 0xF, 4, 4), bit32.lrotate(1, 1), bit32.rrotate(2, 1))
