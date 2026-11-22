-- Runs a script with every unknown global recorded, and prints what it did and returned.
local path = os.getenv("SCRIPT")
local log = {}
local function show(v, depth)
    depth = depth or 0
    if type(v) == "table" then
        if depth > 6 then return "{...}" end
        local keys = {}
        for k in pairs(v) do keys[#keys + 1] = k end
        table.sort(keys, function(a, b) return tostring(a) < tostring(b) end)
        local parts = {}
        for _, k in ipairs(keys) do parts[#parts + 1] = tostring(k) .. "=" .. show(v[k], depth + 1) end
        return "{" .. table.concat(parts, ",") .. "}"
    elseif type(v) == "function" then return "<fn>"
    else return tostring(v) end
end
local counter = 0
local function recorder(name)
    return setmetatable({}, {
        __call = function(_, ...)
            local a = { ... }
            local parts = {}
            for i = 1, select("#", ...) do parts[i] = show(a[i]) end
            log[#log + 1] = name .. "(" .. table.concat(parts, ";") .. ")"
            counter = counter + 1
            if name == "on_ground" or name == "rail_point" then return { 1, 2, 3 } end
            if name == "active" or name == "save.boolean" then return true end
            return counter
        end,
        __index = function(_, k) return recorder(name .. "." .. k) end,
    })
end
local known = {}
for k in pairs(_G) do known[k] = true end
setmetatable(_G, { __index = function(_, k) local r = recorder(k); rawset(_G, k, r); return r end })
rawset(_G, "include", function() end)
local chunk, err = loadfile(path)
if not chunk then print("LOAD ERROR " .. err) return end
local ok, result = pcall(chunk)
print("ok=" .. tostring(ok) .. " result=" .. (ok and show(result) or tostring(result):gsub("^[^:]*:%d+: ", "")))
-- call any functions the script defined, so their bodies run too
local defined = {}
for k, v in pairs(_G) do if not known[k] and type(v) == "function" and k ~= "include" then defined[#defined + 1] = k end end
table.sort(defined)
for _, k in ipairs(defined) do
    local ok2, r = pcall(_G[k], 1, 2, 3)
    log[#log + 1] = "CALL " .. k .. " -> " .. tostring(ok2) .. " " .. (ok2 and show(r) or tostring(r):gsub("^[^:]*:%d+: ", ""))
end
for _, l in ipairs(log) do print(l) end
