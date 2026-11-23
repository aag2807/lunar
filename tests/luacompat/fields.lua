-- Fields of tables made without annotations, as Lua code uses them:
-- cleared with nil, set to nil before they get a value, and functions
-- defined into a table made with {} and then replaced or removed.
local T = { n = "t", f = function(a) return a end }
T.n = nil
T.f = nil
print(T.n, T.f)

local cache = { size = 0 }
cache.last = nil
cache.last = 5
print(cache.last)
cache.last = "five"
print(cache.last)

local M = {}
function M.add(a, b) return a + b end
function M:twice(x) return self.add(x, x) end
print(M.add(1, 2), M:twice(4), M.add(1, 2, 3))

-- Wrapped, as a hook or a profiler does.
local add = M.add
M.add = function(...) return add(...) * 10 end
print(M.add(1, 2))

-- Silenced: Lua drops the arguments a function does not take.
function M.log(level, msg) return level .. ": " .. msg end
print(M.log("info", "up"))
M.log = function() return "quiet" end
print(M.log("info", "down"))

local f = function(a, b) return a + b end
f = function(...) return select("#", ...) end
print(f(1, 2))
f = function(a) return a end
print(f(7, 8))

M.add = nil
print(M.add, M.missing)

local hooks = {}
hooks.onDeath = nil
hooks.onDeath = function(who) return who .. " died" end
print(hooks.onDeath("imp"))
hooks.onDeath = nil
print(hooks.onDeath)

-- A module whose functions call each other before and after they exist.
local Q = {}
function Q.first(n) return Q.second(n) + 1 end
function Q.second(n) return n * 2 end
print(Q.first(3))
