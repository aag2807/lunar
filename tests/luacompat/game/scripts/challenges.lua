-- Challenges: small tasks set out in the open zone.
--
-- Nothing here is built into the game. Each kind of challenge is a Lua
-- function made from things every level script already has: triggers
-- to step on, events to listen for, time.after to wait, set_active to
-- hide and show, a clock on the screen (countdown), and reward(), which
-- pays skill points the first time a challenge is beaten in a save file
-- and nothing after that. include("challenges") in a level, then call:
--
--   ring_run   { name, label, at, rings, seconds, points }
--   dash_to    { name, label, from, to, seconds, points }
--   clear_out  { name, label, at, enemies, seconds, points }
--
-- `name` becomes a key in the save file: small letters, digits and _.
-- Places are { x, z }; everything is stood on the ground.

-- What every challenge shares: one may run at a time, a clock, and an
-- end that is either a win or the clock running out.
local running = nil -- the challenge in progress, or nil

local function finish(c, won)
    running = nil
    countdown()
    if won then
        local paid = reward(c.name, c.points)
        if paid > 0 then
            message(c.label .. ": done")
        else
            message(c.label .. ": done again")
        end
        events.emit("challenge.won")
    else
        message(c.label .. ": out of time")
        events.emit("challenge.lost")
        if c.on_lost then c.on_lost() end
    end
end

local function begin(c)
    if running then return false end
    running = c
    c.attempt = (c.attempt or 0) + 1
    local attempt = c.attempt
    countdown(c.seconds, c.label)
    events.emit("challenge.started")
    -- The timer cannot be taken back, so when it goes off it checks
    -- that the attempt it was set for is still the one in progress.
    time.after(c.seconds, function()
        if running == c and c.attempt == attempt then
            finish(c, false)
        end
    end)
    return true
end

local function start_pad(c, x, z)
    trigger {
        at = on_ground(x, z, 0.3),
        radius = 1.2,
        on_touch = function()
            if running == c then return end
            if running then
                message("finish " .. running.label .. " first")
            elseif c.on_start() ~= false and begin(c) then
                message(c.label .. ": go")
            end
        end,
    }
end

-- Collect every ring of a trail before the clock runs out.
function ring_run(c)
    local set = {}
    for i, place in ipairs(c.rings) do
        local p = on_ground(place[1], place[2], 1.1)
        set[i] = ring { p[1], p[2], p[3] }
        set_active(set[i], false) -- not there until the challenge starts
    end
    c.on_start = function()
        for _, r in ipairs(set) do set_active(r, true) end
    end
    c.on_lost = function()
        for _, r in ipairs(set) do set_active(r, false) end
    end
    events.on("ring.collected", function()
        if running ~= c then return end
        for _, r in ipairs(set) do
            if active(r) then return end
        end
        finish(c, true)
    end)
    start_pad(c, c.at[1], c.at[2])
end

-- Get from one pad to another before the clock runs out.
function dash_to(c)
    c.on_start = function() end
    start_pad(c, c.from[1], c.from[2])
    trigger {
        at = on_ground(c.to[1], c.to[2], 0.3),
        radius = 2.0,
        on_touch = function()
            if running == c then finish(c, true) end
        end,
    }
end

-- Beat every enemy of a group before the clock runs out. `enemies` is
-- a list of what enemy{...} returned.
function clear_out(c)
    c.on_start = function()
        for _, e in ipairs(c.enemies) do
            if active(e) then return end
        end
        message(c.label .. ": nobody left to fight")
        return false
    end
    events.on("enemy.defeated", function()
        if running ~= c then return end
        for _, e in ipairs(c.enemies) do
            if active(e) then return end
        end
        finish(c, true)
    end)
    start_pad(c, c.at[1], c.at[2])
end
