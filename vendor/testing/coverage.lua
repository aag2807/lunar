-- Lunar Code Coverage Module
-- Tracks which lines of code are executed during tests

local coverage = {}

-- Coverage data: file -> line -> hit count
local coverageData = {}
local activeFiles = {}

-- Track if coverage is enabled
local enabled = false

-- Initialize coverage for a file
function coverage.init(filename)
    if not coverageData[filename] then
        coverageData[filename] = {}
        table.insert(activeFiles, filename)
    end
end

-- Record a line hit
function coverage.hit(filename, line)
    if not enabled then
        return
    end

    if not coverageData[filename] then
        coverage.init(filename)
    end

    coverageData[filename][line] = (coverageData[filename][line] or 0) + 1
end

-- Enable coverage tracking
function coverage.start()
    enabled = true
    coverageData = {}
    activeFiles = {}
end

-- Disable coverage tracking
function coverage.stop()
    enabled = false
end

-- Get coverage data
function coverage.getData()
    return coverageData
end

-- Calculate coverage statistics for a file
function coverage.getFileStats(filename, totalLines)
    local data = coverageData[filename] or {}
    local covered = 0

    for line = 1, totalLines do
        if data[line] and data[line] > 0 then
            covered = covered + 1
        end
    end

    return {
        total = totalLines,
        covered = covered,
        percentage = totalLines > 0 and (covered / totalLines * 100) or 0
    }
end

-- Print coverage report
function coverage.report()
    if not enabled and next(coverageData) == nil then
        print("\nNo coverage data collected")
        return
    end

    -- Colors for terminal output
    local colors = {
        red = "\27[31m",
        green = "\27[32m",
        yellow = "\27[33m",
        cyan = "\27[36m",
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

    print("\n" .. string.rep("=", 60))
    print(color("cyan", "Code Coverage Report"))
    print(string.rep("=", 60))

    local totalLines = 0
    local totalCovered = 0
    local fileCount = 0

    -- Calculate totals
    for _, filename in ipairs(activeFiles) do
        local data = coverageData[filename] or {}
        local maxLine = 0

        -- Find max line number
        for line, _ in pairs(data) do
            if line > maxLine then
                maxLine = line
            end
        end

        if maxLine > 0 then
            fileCount = fileCount + 1
            local stats = coverage.getFileStats(filename, maxLine)
            totalLines = totalLines + stats.total
            totalCovered = totalCovered + stats.covered

            -- Print file stats
            local percentage = stats.percentage
            local percentStr = string.format("%.1f%%", percentage)
            local colorName = percentage >= 80 and "green" or percentage >= 50 and "yellow" or "red"

            print(string.format("\n%s", filename))
            print(string.format("  Lines: %d/%d covered (%s)",
                stats.covered,
                stats.total,
                color(colorName, percentStr)
            ))

            -- Show uncovered lines
            local uncovered = {}
            for line = 1, stats.total do
                if not data[line] or data[line] == 0 then
                    table.insert(uncovered, line)
                end
            end

            if #uncovered > 0 and #uncovered <= 10 then
                print("  Uncovered lines: " .. table.concat(uncovered, ", "))
            elseif #uncovered > 10 then
                local first5 = {}
                for i = 1, 5 do
                    table.insert(first5, uncovered[i])
                end
                print("  Uncovered lines: " .. table.concat(first5, ", ") .. " ... (+" .. (#uncovered - 5) .. " more)")
            end
        end
    end

    -- Print overall stats
    print("\n" .. string.rep("-", 60))
    local overallPercentage = totalLines > 0 and (totalCovered / totalLines * 100) or 0
    local percentStr = string.format("%.1f%%", overallPercentage)
    local colorName = overallPercentage >= 80 and "green" or overallPercentage >= 50 and "yellow" or "red"

    print(string.format("Total Coverage: %d/%d lines (%s)",
        totalCovered,
        totalLines,
        color(colorName, percentStr)
    ))
    print(string.format("Files: %d", fileCount))
    print(string.rep("=", 60))
end

-- Reset coverage data
function coverage.reset()
    coverageData = {}
    activeFiles = {}
end

return coverage
