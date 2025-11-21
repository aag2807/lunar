-- Lunar Testing Library
-- A simple but powerful testing framework

local testing = {}

-- Internal state
local currentSuite = nil
local suites = {}
local beforeEachFns = {}
local afterEachFns = {}
local beforeAllFns = {}
local afterAllFns = {}

-- Colors for terminal output
local colors = {
    red = "\27[31m",
    green = "\27[32m",
    yellow = "\27[33m",
    reset = "\27[0m"
}

-- Check if running in terminal with color support
local useColors = os.getenv("TERM") ~= nil

local function color(name, text)
    if useColors then
        return colors[name] .. text .. colors.reset
    end
    return text
end

-- Expectation builder
local function createExpectation(value)
    local exp = {}

    function exp.toBe(expected)
        if value ~= expected then
            error(string.format("Expected %s to be %s", tostring(value), tostring(expected)))
        end
    end

    function exp.toEqual(expected)
        -- Deep equality check
        if type(value) == "table" and type(expected) == "table" then
            for k, v in pairs(expected) do
                if value[k] ~= v then
                    error(string.format("Expected %s to equal %s", tostring(value), tostring(expected)))
                end
            end
            for k, v in pairs(value) do
                if expected[k] ~= v then
                    error(string.format("Expected %s to equal %s", tostring(value), tostring(expected)))
                end
            end
        elseif value ~= expected then
            error(string.format("Expected %s to equal %s", tostring(value), tostring(expected)))
        end
    end

    function exp.toBeTruthy()
        if not value then
            error(string.format("Expected %s to be truthy", tostring(value)))
        end
    end

    function exp.toBeFalsy()
        if value then
            error(string.format("Expected %s to be falsy", tostring(value)))
        end
    end

    function exp.toBeNil()
        if value ~= nil then
            error(string.format("Expected %s to be nil", tostring(value)))
        end
    end

    function exp.toBeGreaterThan(expected)
        if value <= expected then
            error(string.format("Expected %s to be greater than %s", tostring(value), tostring(expected)))
        end
    end

    function exp.toBeLessThan(expected)
        if value >= expected then
            error(string.format("Expected %s to be less than %s", tostring(value), tostring(expected)))
        end
    end

    function exp.toContain(expected)
        if type(value) == "string" then
            if not string.find(value, expected, 1, true) then
                error(string.format("Expected '%s' to contain '%s'", value, expected))
            end
        elseif type(value) == "table" then
            local found = false
            for _, v in ipairs(value) do
                if v == expected then
                    found = true
                    break
                end
            end
            if not found then
                error(string.format("Expected table to contain %s", tostring(expected)))
            end
        end
    end

    function exp.toThrow()
        if type(value) ~= "function" then
            error("Expected a function for toThrow()")
        end
        local success = pcall(value)
        if success then
            error("Expected function to throw an error")
        end
    end

    return exp
end

-- Public API

function testing.describe(name, fn)
    local suite = {
        name = name,
        tests = {},
        beforeEach = {},
        afterEach = {}
    }

    local previousSuite = currentSuite
    currentSuite = suite

    -- Copy current lifecycle hooks
    for _, hook in ipairs(beforeEachFns) do
        table.insert(suite.beforeEach, hook)
    end
    for _, hook in ipairs(afterEachFns) do
        table.insert(suite.afterEach, hook)
    end

    fn()

    table.insert(suites, suite)
    currentSuite = previousSuite
end

function testing.it(name, fn)
    if not currentSuite then
        error("it() must be called inside describe()")
    end
    table.insert(currentSuite.tests, {name = name, fn = fn})
end

testing.test = testing.it

function testing.expect(value)
    return createExpectation(value)
end

function testing.assert(condition, message)
    if not condition then
        error(message or "Assertion failed")
    end
end

function testing.assertEqual(actual, expected, message)
    if actual ~= expected then
        error(message or string.format("Expected %s but got %s", tostring(expected), tostring(actual)))
    end
end

function testing.assertNil(value, message)
    if value ~= nil then
        error(message or string.format("Expected nil but got %s", tostring(value)))
    end
end

function testing.assertNotNil(value, message)
    if value == nil then
        error(message or "Expected non-nil value")
    end
end

function testing.assertError(fn, message)
    local success = pcall(fn)
    if success then
        error(message or "Expected function to throw an error")
    end
end

function testing.beforeEach(fn)
    if currentSuite then
        table.insert(currentSuite.beforeEach, fn)
    else
        table.insert(beforeEachFns, fn)
    end
end

function testing.afterEach(fn)
    if currentSuite then
        table.insert(currentSuite.afterEach, fn)
    else
        table.insert(afterEachFns, fn)
    end
end

function testing.beforeAll(fn)
    table.insert(beforeAllFns, fn)
end

function testing.afterAll(fn)
    table.insert(afterAllFns, fn)
end

function testing.runTests()
    local startTime = os.clock()
    local ctx = {
        name = "Test Results",
        passed = 0,
        failed = 0,
        errors = {},
        startTime = startTime,
        suiteCount = 0,
        testCount = 0
    }

    -- Run beforeAll hooks
    for _, hook in ipairs(beforeAllFns) do
        hook()
    end

    -- Run all suites
    for _, suite in ipairs(suites) do
        ctx.suiteCount = ctx.suiteCount + 1
        print(string.format("\n%s", suite.name))

        for _, test in ipairs(suite.tests) do
            ctx.testCount = ctx.testCount + 1

            -- Run beforeEach hooks
            for _, hook in ipairs(suite.beforeEach) do
                hook()
            end

            -- Run test with timing
            local testStartTime = os.clock()
            local success, err = pcall(test.fn)
            local testDuration = os.clock() - testStartTime

            if success then
                ctx.passed = ctx.passed + 1
                local durationStr = testDuration > 0.001 and string.format(" (%.0fms)", testDuration * 1000) or ""
                print(string.format("  %s %s%s", color("green", "✓"), test.name, color("reset", durationStr)))
            else
                ctx.failed = ctx.failed + 1
                print(string.format("  %s %s", color("red", "✗"), test.name))
                print(string.format("    %s", color("red", tostring(err))))
                table.insert(ctx.errors, {suite = suite.name, test = test.name, error = err})
            end

            -- Run afterEach hooks
            for _, hook in ipairs(suite.afterEach) do
                hook()
            end
        end
    end

    -- Run afterAll hooks
    for _, hook in ipairs(afterAllFns) do
        hook()
    end

    -- Calculate total duration
    ctx.duration = os.clock() - startTime

    -- Reset state for next run
    suites = {}
    beforeEachFns = {}
    afterEachFns = {}
    beforeAllFns = {}
    afterAllFns = {}

    return ctx
end

function testing.printResults(ctx)
    print(string.format("\n%s", string.rep("=", 50)))

    -- Test summary
    local total = ctx.passed + ctx.failed
    local passRate = total > 0 and string.format("%.1f%%", (ctx.passed / total) * 100) or "0%"

    if ctx.failed == 0 then
        print(string.format("%s All tests passed! %s", color("green", "✓"), color("green", passRate)))
    else
        print(string.format("%s Some tests failed", color("red", "✗")))
    end

    print(string.format("\nSuites:  %d passed, %d total", ctx.suiteCount, ctx.suiteCount))
    print(string.format("Tests:   %s passed, %s failed, %d total",
        color("green", tostring(ctx.passed)),
        ctx.failed > 0 and color("red", tostring(ctx.failed)) or tostring(ctx.failed),
        total
    ))

    -- Duration
    if ctx.duration then
        local durationMs = ctx.duration * 1000
        local timeStr = durationMs >= 1000
            and string.format("%.2fs", ctx.duration)
            or string.format("%.0fms", durationMs)
        print(string.format("Time:    %s", timeStr))
    end

    -- Failed tests summary
    if ctx.failed > 0 then
        print(color("red", "\nFailed Tests:"))
        for _, err in ipairs(ctx.errors) do
            print(string.format("  %s %s > %s", color("red", "✗"), err.suite, err.test))
        end
    end

    print(string.rep("=", 50))
end

return testing
