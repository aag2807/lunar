-- Example Lua file to migrate to Lunar

-- Simple variables
local name = "Alice"
local age = 30
local isActive = true
local items = {1, 2, 3, 4, 5}

-- Function with basic types
function greet(person)
    return "Hello, " .. person
end

-- Function with number operations
function add(a, b)
    return a + b
end

-- Function with table
function processData(data)
    local result = {}
    for i, value in ipairs(data) do
        result[i] = value * 2
    end
    return result
end

-- Conditional logic
function checkAge(person_age)
    if person_age >= 18 then
        return "adult"
    else
        return "minor"
    end
end

-- Loop examples
function sum(numbers)
    local total = 0
    for i = 1, #numbers do
        total = total + numbers[i]
    end
    return total
end

-- Table operations
function createPerson(name, age)
    return {
        name = name,
        age = age,
        greet = function()
            print("Hello, I'm " .. name)
        end
    }
end

-- Main logic
print(greet(name))
print("Age check:", checkAge(age))
print("Sum:", sum(items))

local person = createPerson("Bob", 25)
person.greet()
