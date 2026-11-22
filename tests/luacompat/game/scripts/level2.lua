-- Level 2: the canyon.
--
-- Where things are is kept in level2_objects.lua, which the in-game
-- editor writes. Press 2 and then Tab in the game to change it.
include("level2_objects")

-- These rings stay here, written by hand, because they follow the
-- rails: move a rail in level.c and they move with it. The editor
-- cannot select them. That is the trade from chapter 11, made the
-- other way round this time.
for r = 1, rail_count() do
    local t = 0.5
    while t < rail_length(r) do
        local p = rail_point(r, t)
        ring { p[1], p[2] + 0.8, p[3] }
        t = t + 0.5
    end
end

-- The goal stands on the mesa at the far end, and is not there at all
-- until you are carrying enough rings.
local gate = goal(on_ground(0, -32, 2.0))
local price = 10
local open = false
set_active(gate, false)

events.on("ring.collected", function(count)
    if not open and count >= price then
        open = true
        set_active(gate, true)
        message("the goal on the mesa is open")
    end
end)

-- A pad in front of the start that says what to do.
trigger {
    at = on_ground(0, 38, 0.3),
    radius = 0.9,
    on_touch = function()
        if open then
            message("the goal is open: head for the mesa")
        else
            message(string.format("%d rings open the goal", price))
        end
    end,
}
