-- Escapes, long strings, long comments and numerals, read as Lua reads them.
print(#"\65\066", "\65\066" == "AB", "\97" == "a", #"\a", ("\a"):byte())
print(#"a\z
      b", "a\z   b")
print(("\xff"):byte(), ("\x001"):byte(1, -1))
print(("\0001"):byte(1, -1))
print(#"\u{7FF}", #"\u{FFFF}", #"\u{10FFFF}", ("\u{D800}"):byte(1, -1))
local s = "tab\tnl\\n quote\" apos\' back\\"
print(s, #s)
print("line\
two")
local pat = '[%z\1-\31\\"]'
print(#pat, pat:byte(1, -1))
print(("x\1y\31z"):gsub(pat, "?"))
local long = [[
first line kept, leading newline dropped]]
print(#long, long)
local lv = [==[
a]]b]=]c]==]
print(lv)
--[=[
print("commented out")
--]=]
print("after level-1 comment") --[==[ inline ]==] print("same line")
--[[ a ]] print("after short long comment")
---[[ three dashes: a line comment
print("three dashes is a line comment")
--]]
print(4., 2.E-1, 0xA., 0x.8p1, 3.e2, 1e309, -1e309, 0x10, 0xff // 1)
print(.5, 5e-1, 0X1P4, math.type(4.), math.type(0xA.), math.type(0x10))
--[==[
a level-2 comment ]] with ]=] inside
]==]
local l3 = [===[
level ]==] three]===]
print(l3)
