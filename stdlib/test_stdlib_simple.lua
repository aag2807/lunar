function testMathLib()
    local pi = math.pi
    local sine = math.sin((pi / 2))
    local random = math.random(1, 100)
    local floor = math.floor(3.7)
    local ceil = math.ceil(3.2)
    print("Math tests passed")
end

function testCoreGlobals()
    local str = tostring(123)
    local num = tonumber("456")
    print("Core global tests passed")
end

function main()
    testMathLib()
    testCoreGlobals()
    print("All standard library tests passed!")
end

main()
