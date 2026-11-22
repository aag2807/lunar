-- Lunar's own keywords are ordinary names in Lua.
local function enemy(t)
    return t.from[1] + t.to[1] + t.speed
end
print(enemy { from = { -6, 0.6, 10 }, to = { -1, 0.6, 10 }, speed = 2.0 })

local set = {}
set[1] = 5
local get = function(t, k) return t[k] end
print(get(set, 1))

local match, with, is, as = 1, 2, 3, 4
print(match + with + is + as)
local class, enum, interface, namespace, const, readonly = "c", "e", "i", "n", "k", "r"
print(class .. enum .. interface .. namespace .. const .. readonly)
local declare, import, export, async, await, abstract, static = 1, 2, 3, 4, 5, 6, 7
print(declare + import + export + async + await + abstract + static)
local extends, implements, keyof, typeof, constructor, super = 1, 2, 3, 4, 5, 6
local public, private, protected, number, from, type = 7, 8, 9, 10, 11, 12
print(extends + implements + keyof + typeof + constructor + super + public + private + protected + number + from + type)

local t = { from = 1, match = 2, class = 3, self = 4, number = 5, set = 6, get = 7, type = 8, import = 9, with = 10 }
print(t.from, t.match, t.class, t.self, t.number, t.set, t.get, t.type, t.import, t.with)

local s = "abc123"
print(s:match("%d+"), ("x1"):match("%d"), string.match(s, "%a+"))
local obj = { match = function(self, p) return "m:" .. p end }
print(obj:match("z"))

local function f(from, to, match, number, set, get, class, as)
    return from + to + match + number + set + get + class + as
end
print(f(1, 2, 3, 4, 5, 6, 7, 8))
for from, to in ipairs({ 10, 20 }) do
    print(from, to)
end
