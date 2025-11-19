-- Lunar HTTP Client Library
-- A simple HTTP client built on LuaSocket

local http = {}

-- Try to load LuaSocket
local socket_http, ltn12, url_parser
local hasSocket = pcall(function()
    socket_http = require("socket.http")
    ltn12 = require("ltn12")
    url_parser = require("socket.url")
end)

if not hasSocket then
    -- Provide stub functions with helpful error
    local function noSocket()
        error("HTTP library requires LuaSocket. Install with: luarocks install luasocket")
    end

    http.get = noSocket
    http.post = noSocket
    http.put = noSocket
    http.delete = noSocket
    http.request = noSocket

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

-- Parse a URL into components
function http.parseURL(url)
    return url_parser.parse(url)
end

-- Generic HTTP request
function http.request(method, url, options)
    options = options or {}

    local responseBody = {}
    local requestBody = options.body

    local requestHeaders = options.headers or {}

    -- Set content length for body
    if requestBody then
        requestHeaders["Content-Length"] = #requestBody
        if not requestHeaders["Content-Type"] then
            requestHeaders["Content-Type"] = "application/x-www-form-urlencoded"
        end
    end

    local response, status, headers = socket_http.request({
        url = url,
        method = method,
        headers = requestHeaders,
        source = requestBody and ltn12.source.string(requestBody) or nil,
        sink = ltn12.sink.table(responseBody),
        timeout = options.timeout or 30
    })

    return {
        status = status,
        body = table.concat(responseBody),
        headers = headers or {}
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

return http
