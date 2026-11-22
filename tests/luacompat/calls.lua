-- Calls written with a table or a string instead of parentheses.
for _, place in ipairs { { -12, 197 }, { -15, 200 } } do
    print(place[1], place[2])
end
print(type "x", tostring "y")
local function wrap(t)
    return function(u) return #t + #u end
end
print(wrap { 1, 2 } { 3 })
local obj = { tag = function(self, t) return t[1] end }
print(obj:tag { "first" }, obj:tag "s")
package.preload["shapes"] = function() return { sides = 4 } end
print(require "shapes".sides, (require "shapes").sides)

-- A parenthesised expression that starts with a method call.
local s = "abc"
local n = (s:len() + 1) * 2
print(n, (s:upper()), (s):rep(2))

-- Parentheses keep only the first value of a call.
local function two() return 1, 2 end
print((two()))
print(two())
local packed = { (two()) }
print(#packed, select("#", (two())))
