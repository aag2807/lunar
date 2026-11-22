-- Classes built from metatables: constructors, methods with ':',
-- inheritance through __index, and operator metamethods.
local Animal = {}
Animal.__index = Animal

function Animal.new(name, sound)
    local self = setmetatable({}, Animal)
    self.name = name
    self.sound = sound
    return self
end

function Animal:speak()
    return self.name .. " says " .. self.sound
end

local Dog = setmetatable({}, { __index = Animal })
Dog.__index = Dog

function Dog.new(name)
    local self = Animal.new(name, "woof")
    return setmetatable(self, Dog)
end

function Dog:fetch()
    return self.name .. " fetches"
end

local d = Dog.new("rex")
print(d:speak(), d:fetch(), getmetatable(d) == Dog)

local Vec = {}
Vec.__index = Vec
local function vec(x, y) return setmetatable({ x = x, y = y }, Vec) end
Vec.__add = function(p, q) return vec(p.x + q.x, p.y + q.y) end
Vec.__eq = function(p, q) return p.x == q.x and p.y == q.y end
Vec.__lt = function(p, q) return p:len() < q:len() end
Vec.__tostring = function(p) return "(" .. p.x .. ", " .. p.y .. ")" end
Vec.__len = function() return 2 end
function Vec:len() return math.sqrt(self.x * self.x + self.y * self.y) end

local s = vec(1, 2) + vec(3, 4)
print(tostring(s), s == vec(4, 6), vec(1, 1) < vec(3, 3), #s, s:len())

local Stack = {}
Stack.__index = Stack
function Stack.new() return setmetatable({ items = {}, size = 0 }, Stack) end
function Stack:push(v)
    self.size = self.size + 1
    self.items[self.size] = v
end
function Stack:pop()
    local v = self.items[self.size]
    self.items[self.size] = nil
    self.size = self.size - 1
    return v
end
local st = Stack.new()
st:push(1)
st:push(2)
print(st:pop(), st:pop(), st.size)

local defaults = setmetatable({}, { __index = function(_, k) return k .. "?" end })
local counted = setmetatable({}, { __newindex = function(tbl, k, v) rawset(tbl, k, v * 10) end })
counted.n = 4
print(defaults.color, counted.n, rawget(counted, "n"), rawlen({ 1, 2 }))
