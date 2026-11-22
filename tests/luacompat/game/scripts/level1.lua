-- The level, in two parts.
--
-- Where things are is kept in level1_objects.lua, which the in-game
-- editor writes. Press Tab in the game to change it.
include("level1_objects")

-- What the level does is written here by hand, and the editor never
-- touches it.

local up = { 0, 1, 0 }

-- The spring beside the tall platform starts switched off. Collect
-- enough rings and it appears.
local lift = spring { at = on_ground(10, 20.5, 0.2), dir = up, speed = 16 }
local lift_price = 8
local lift_open = false
set_active(lift, false)

local function open_lift()
    lift_open = true
    set_active(lift, true)
end

-- Once opened, it stays open: the `save` service remembers it, for as
-- long as the game is running. (Chapter 9 makes that "for good".)
if save.get("level1.lift_open", 0) == 1 then
    open_lift()
end

-- "ring.collected" is an event the game sends whenever a ring is picked
-- up, with how many are now held. This script is one of its listeners;
-- the sound of the ring is another.
events.on("ring.collected", function(count)
    if not lift_open and count >= lift_price then
        open_lift()
        save.set("level1.lift_open", 1)
        message("the spring by the platform is open")
    end
end)

-- Being hit stops the world for a moment: time runs at a third of its
-- speed for a quarter of a (slowed) second, and then picks up again.
events.on("player.hurt", function()
    time.scale(0.35)
    time.after(0.25, function() time.scale(1) end)
end)

-- The third enemy beaten is worth a prize, and a sound of its own.
events.on("enemy.defeated", function(count)
    if count == 3 then
        add_rings(5)
        audio.play("ring", 0.8, 2.0)
        message("three down: five rings")
    end
end)

-- A pad in front of the start. Step on it for a hint, and a boost if
-- you are short of rings.
trigger {
    at = on_ground(0, 17, 0.3),
    radius = 0.9,
    on_touch = function()
        if lift_open then
            message("go and find that spring")
        elseif rings() == 0 then
            add_rings(3)
            message("three rings to get you started")
        else
            message(string.format("%d more rings open the spring", lift_price - rings()))
        end
    end,
}
