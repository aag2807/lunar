-- Lunar HTTP Client Library
-- Uses curl for HTTP requests (no external dependencies)

local http = {}

-- Check if curl is available
local function hasCurl()
    local handle = io.popen("curl --version 2>/dev/null")
    if handle then
        local result = handle:read("*a")
        handle:close()
        return result and #result > 0
    end
    return false
end

if not hasCurl() then
    local function noCurl()
        error("HTTP library requires curl. Please install curl.")
    end

    http.get = noCurl
    http.post = noCurl
    http.put = noCurl
    http.delete = noCurl
    http.request = noCurl

    return http
end

-- URL encode a string
function http.encodeURI(str)
    if str then
        str = string.gsub(str, "\n", "\r\n")
        str = string.gsub(str, "([^%w %-%_%.%~])", function(c)
            return string.format("%%%02X", string.byte(c))
        end)
        str = string.gsub(str, " ", "+")
    end
    return str
end

-- URL decode a string
function http.decodeURI(str)
    str = string.gsub(str, "+", " ")
    str = string.gsub(str, "%%(%x%x)", function(h)
        return string.char(tonumber(h, 16))
    end)
    return str
end

-- Escape string for shell
local function shellEscape(str)
    return "'" .. string.gsub(str, "'", "'\\''") .. "'"
end

-- Generic HTTP request using curl
function http.request(method, url, options)
    options = options or {}

    -- Build curl command
    local cmd = "curl -s -w '\\n%{http_code}'"

    -- Method
    cmd = cmd .. " -X " .. method

    -- Timeout
    local timeout = options.timeout or 30
    cmd = cmd .. " --max-time " .. timeout

    -- Headers
    local headers = options.headers or {}
    for key, value in pairs(headers) do
        cmd = cmd .. " -H " .. shellEscape(key .. ": " .. value)
    end

    -- Body
    if options.body then
        -- Set default content type if not specified
        if not headers["Content-Type"] and not headers["content-type"] then
            cmd = cmd .. " -H 'Content-Type: application/x-www-form-urlencoded'"
        end
        cmd = cmd .. " -d " .. shellEscape(options.body)
    end

    -- URL
    cmd = cmd .. " " .. shellEscape(url)

    -- Execute
    local handle = io.popen(cmd .. " 2>/dev/null")
    if not handle then
        return {
            status = 0,
            body = "",
            headers = {},
            error = "Failed to execute curl"
        }
    end

    local output = handle:read("*a")
    handle:close()

    -- Parse response - status code is on the last line
    local body, status = "", 0
    local lastNewline = output:match(".*\n()")
    if lastNewline then
        body = output:sub(1, lastNewline - 2)  -- Remove trailing newline and status
        status = tonumber(output:sub(lastNewline)) or 0
    else
        -- No newline found, entire output might be status code
        status = tonumber(output) or 0
    end

    return {
        status = status,
        body = body,
        headers = {}  -- curl -s doesn't return headers by default
    }
end

-- GET request
function http.get(url, options)
    return http.request("GET", url, options)
end

-- POST request
function http.post(url, body, options)
    options = options or {}
    options.body = body
    return http.request("POST", url, options)
end

-- PUT request
function http.put(url, body, options)
    options = options or {}
    options.body = body
    return http.request("PUT", url, options)
end

-- DELETE request
function http.delete(url, options)
    return http.request("DELETE", url, options)
end

-- HEAD request
function http.head(url, options)
    return http.request("HEAD", url, options)
end

-- PATCH request
function http.patch(url, body, options)
    options = options or {}
    options.body = body
    return http.request("PATCH", url, options)
end

-- Helper to make JSON requests
function http.json(method, url, data, options)
    options = options or {}
    options.headers = options.headers or {}
    options.headers["Content-Type"] = "application/json"
    options.headers["Accept"] = "application/json"

    if data and type(data) == "table" then
        -- Simple JSON encoding for tables
        local json = require("vendor/json") or { encode = function(t) return tostring(t) end }
        options.body = json.encode(data)
    elseif data then
        options.body = data
    end

    return http.request(method, url, options)
end

return http
