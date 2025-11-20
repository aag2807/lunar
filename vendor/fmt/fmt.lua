-- Lunar Format Library
-- Provides formatted output functions

local fmt = {}

-- Print formatted string to stdout
function fmt.printf(format, ...)
    io.write(string.format(format, ...))
end

-- Return formatted string
function fmt.sprintf(format, ...)
    return string.format(format, ...)
end

-- Print with newline
function fmt.println(...)
    local args = {...}
    local parts = {}
    for i = 1, select('#', ...) do
        parts[i] = tostring(args[i])
    end
    print(table.concat(parts, "\t"))
end

-- Print without newline
function fmt.print(...)
    local args = {...}
    local parts = {}
    for i = 1, select('#', ...) do
        parts[i] = tostring(args[i])
    end
    io.write(table.concat(parts, "\t"))
end

-- Format a value for debugging (like inspect/pretty-print)
function fmt.inspect(value, depth)
    depth = depth or 3

    local function _inspect(val, currentDepth, visited)
        if currentDepth > depth then
            return "..."
        end

        local valType = type(val)

        if valType == "nil" then
            return "nil"
        elseif valType == "boolean" then
            return tostring(val)
        elseif valType == "number" then
            return tostring(val)
        elseif valType == "string" then
            return string.format("%q", val)
        elseif valType == "function" then
            return "<function>"
        elseif valType == "userdata" then
            return "<userdata>"
        elseif valType == "thread" then
            return "<thread>"
        elseif valType == "table" then
            -- Check for cycles
            if visited[val] then
                return "<circular>"
            end
            visited[val] = true

            local parts = {}
            local arrayPart = {}
            local hashPart = {}

            -- Separate array and hash parts
            local maxIndex = 0
            for k, v in pairs(val) do
                if type(k) == "number" and k > 0 and k == math.floor(k) then
                    maxIndex = math.max(maxIndex, k)
                    arrayPart[k] = v
                else
                    hashPart[k] = v
                end
            end

            -- Format array part
            for i = 1, maxIndex do
                if arrayPart[i] ~= nil then
                    table.insert(parts, _inspect(arrayPart[i], currentDepth + 1, visited))
                else
                    table.insert(parts, "nil")
                end
            end

            -- Format hash part
            local keys = {}
            for k in pairs(hashPart) do
                table.insert(keys, k)
            end
            table.sort(keys, function(a, b)
                return tostring(a) < tostring(b)
            end)

            for _, k in ipairs(keys) do
                local keyStr
                if type(k) == "string" and k:match("^[%a_][%w_]*$") then
                    keyStr = k
                else
                    keyStr = "[" .. _inspect(k, currentDepth + 1, visited) .. "]"
                end
                table.insert(parts, keyStr .. " = " .. _inspect(hashPart[k], currentDepth + 1, visited))
            end

            visited[val] = nil

            if #parts == 0 then
                return "{}"
            else
                return "{ " .. table.concat(parts, ", ") .. " }"
            end
        else
            return "<unknown>"
        end
    end

    return _inspect(value, 1, {})
end

-- Error formatting - prints to stderr
function fmt.errorf(format, ...)
    io.stderr:write(string.format(format, ...))
end

return fmt
