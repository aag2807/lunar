-- Lunar JSON Library
-- Pure Lua JSON encoder/decoder

local json = {}

-- Escape special characters in strings
local escapeChars = {
    ["\\"] = "\\\\",
    ["\""] = "\\\"",
    ["\n"] = "\\n",
    ["\r"] = "\\r",
    ["\t"] = "\\t",
    ["\b"] = "\\b",
    ["\f"] = "\\f"
}

local function escapeString(str)
    return str:gsub('[\\"\n\r\t\b\f]', escapeChars)
end

-- Encode a Lua value to JSON
local function encodeValue(value, indent, currentIndent)
    local valueType = type(value)

    if value == nil then
        return "null"
    elseif valueType == "boolean" then
        return value and "true" or "false"
    elseif valueType == "number" then
        if value ~= value then -- NaN
            return "null"
        elseif value == math.huge or value == -math.huge then
            return "null"
        end
        return tostring(value)
    elseif valueType == "string" then
        return '"' .. escapeString(value) .. '"'
    elseif valueType == "table" then
        -- Check if it's an array
        local isArray = true
        local maxIndex = 0
        for k, v in pairs(value) do
            if type(k) ~= "number" or k <= 0 or math.floor(k) ~= k then
                isArray = false
                break
            end
            maxIndex = math.max(maxIndex, k)
        end

        -- Also check for gaps
        if isArray then
            for i = 1, maxIndex do
                if value[i] == nil then
                    isArray = false
                    break
                end
            end
        end

        local parts = {}
        local newIndent = currentIndent and (currentIndent .. indent) or nil

        if isArray then
            for i = 1, maxIndex do
                local encoded = encodeValue(value[i], indent, newIndent)
                if newIndent then
                    table.insert(parts, newIndent .. encoded)
                else
                    table.insert(parts, encoded)
                end
            end
            if newIndent and #parts > 0 then
                return "[\n" .. table.concat(parts, ",\n") .. "\n" .. currentIndent .. "]"
            else
                return "[" .. table.concat(parts, ",") .. "]"
            end
        else
            for k, v in pairs(value) do
                local key = type(k) == "string" and k or tostring(k)
                local encoded = encodeValue(v, indent, newIndent)
                if newIndent then
                    table.insert(parts, newIndent .. '"' .. escapeString(key) .. '": ' .. encoded)
                else
                    table.insert(parts, '"' .. escapeString(key) .. '":' .. encoded)
                end
            end
            if newIndent and #parts > 0 then
                return "{\n" .. table.concat(parts, ",\n") .. "\n" .. currentIndent .. "}"
            else
                return "{" .. table.concat(parts, ",") .. "}"
            end
        end
    else
        error("Cannot encode " .. valueType .. " to JSON")
    end
end

function json.encode(value)
    return encodeValue(value, nil, nil)
end

function json.pretty(value, indentSize)
    indentSize = indentSize or 2
    local indent = string.rep(" ", indentSize)
    return encodeValue(value, indent, "")
end

-- Decode JSON string to Lua value
local function skipWhitespace(str, pos)
    return str:match("^%s*()", pos)
end

local function decodeValue(str, pos)
    pos = skipWhitespace(str, pos)
    local char = str:sub(pos, pos)

    if char == '"' then
        -- String
        local endPos = pos + 1
        while true do
            local c = str:sub(endPos, endPos)
            if c == '"' then
                break
            elseif c == '\\' then
                endPos = endPos + 2
            elseif c == '' then
                error("Unterminated string")
            else
                endPos = endPos + 1
            end
        end
        local s = str:sub(pos + 1, endPos - 1)
        -- Unescape
        s = s:gsub('\\(.)', function(c)
            if c == 'n' then return '\n'
            elseif c == 'r' then return '\r'
            elseif c == 't' then return '\t'
            elseif c == 'b' then return '\b'
            elseif c == 'f' then return '\f'
            else return c
            end
        end)
        return s, endPos + 1
    elseif char == '{' then
        -- Object
        local obj = {}
        pos = pos + 1
        pos = skipWhitespace(str, pos)
        if str:sub(pos, pos) == '}' then
            return obj, pos + 1
        end
        while true do
            pos = skipWhitespace(str, pos)
            local key, newPos = decodeValue(str, pos)
            if type(key) ~= "string" then
                error("Expected string key in object")
            end
            pos = skipWhitespace(str, newPos)
            if str:sub(pos, pos) ~= ':' then
                error("Expected ':' after key")
            end
            pos = pos + 1
            local value
            value, pos = decodeValue(str, pos)
            obj[key] = value
            pos = skipWhitespace(str, pos)
            local c = str:sub(pos, pos)
            if c == '}' then
                return obj, pos + 1
            elseif c ~= ',' then
                error("Expected ',' or '}' in object")
            end
            pos = pos + 1
        end
    elseif char == '[' then
        -- Array
        local arr = {}
        pos = pos + 1
        pos = skipWhitespace(str, pos)
        if str:sub(pos, pos) == ']' then
            return arr, pos + 1
        end
        while true do
            local value
            value, pos = decodeValue(str, pos)
            table.insert(arr, value)
            pos = skipWhitespace(str, pos)
            local c = str:sub(pos, pos)
            if c == ']' then
                return arr, pos + 1
            elseif c ~= ',' then
                error("Expected ',' or ']' in array")
            end
            pos = pos + 1
        end
    elseif str:sub(pos, pos + 3) == "true" then
        return true, pos + 4
    elseif str:sub(pos, pos + 4) == "false" then
        return false, pos + 5
    elseif str:sub(pos, pos + 3) == "null" then
        return nil, pos + 4
    elseif char == '-' or char:match("%d") then
        -- Number
        local numStr = str:match("^-?%d+%.?%d*[eE]?[+-]?%d*", pos)
        return tonumber(numStr), pos + #numStr
    else
        error("Unexpected character: " .. char)
    end
end

function json.decode(str)
    local value, pos = decodeValue(str, 1)
    pos = skipWhitespace(str, pos)
    if pos <= #str then
        error("Unexpected data after JSON")
    end
    return value
end

function json.isValid(str)
    local success = pcall(json.decode, str)
    return success
end

return json
