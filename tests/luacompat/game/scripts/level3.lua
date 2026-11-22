-- Level 3: the open zone.
--
-- Half a kilometre of land each way, from a height map. Where the
-- editor's things are is kept in level3_objects.lua: press 3 and then
-- Tab in the game to add to it.
include("level3_objects")

-- Everything below is placed by asking the land how high it is, so it
-- stays on the ground if the land is repainted.

-- A line of rings from one place to another, a ring every `gap` metres.
local function trail(x0, z0, x1, z1, gap)
    local length = math.sqrt((x1 - x0) ^ 2 + (z1 - z0) ^ 2)
    local count = math.floor(length / gap)
    for i = 0, count do
        local t = i / count
        local p = on_ground(x0 + (x1 - x0) * t, z0 + (z1 - z0) * t, 1.1)
        ring { p[1], p[2], p[3] }
    end
end

-- Out of the start, down into the valley, and up toward the mesa.
trail(0, 180, 0, 110, 7)
trail(4, 96, 30, 30, 8)
trail(34, 16, 60, -70, 9)
trail(62, -84, 70, -138, 9)

-- A handful beside the start to try the loop trail on: boost in a
-- circle round them and close it.
for _, place in ipairs { { -12, 197 }, { -15, 200 }, { -12, 203 }, { -9, 200 }, { -12, 200 } } do
    local p = on_ground(place[1], place[2], 1.1)
    ring { p[1], p[2], p[3] }
end

-- A ring of rings round the rim of the bowl in the west.
for i = 0, 11 do
    local a = i / 12 * 2 * math.pi
    local p = on_ground(-110 + 34 * math.cos(a), 70 + 34 * math.sin(a), 1.1)
    ring { p[1], p[2], p[3] }
end

-- Rings along both rails.
for r = 1, rail_count() do
    local t = 1.0
    while t < rail_length(r) do
        local p = rail_point(r, t)
        ring { p[1], p[2] + 0.8, p[3] }
        t = t + 4.0
    end
end

-- Something to fight on the way: two sentries in the valley, and a
-- brute guarding the climb.
local sentry_west = enemy { from = on_ground(-8, 30, 0.6), to = on_ground(2, 30, 0.6), speed = 2.0 }
local sentry_east = enemy { from = on_ground(40, -20, 0.6), to = on_ground(48, -26, 0.6), speed = 2.5 }
enemy { from = on_ground(62, -120, 0.6), to = on_ground(70, -124, 0.6), speed = 2.0, kind = "brute" }

-- Shards: each raises a stat by a level, once in a save file. What the
-- stats are is in stats.lua; these are only where to look. None of
-- them is on the way to the goal.
local function shards(stat, places)
    for _, place in ipairs(places) do
        shard { at = on_ground(place[1], place[2], 1.2), stat = stat }
    end
end
shards("speed",   { { -110, 70 }, { 150, 120 }, { -180, -150 } })   -- the bowl, and two far corners
shards("force",   { { 44, -23 }, { -60, -40 }, { 120, -60 } })      -- one behind the eastern sentry
shards("stamina", { { 0, 228 }, { -200, 190 }, { 70, -200 } })      -- one behind the start, one behind the mesa

-- Challenges. Each pays a skill point the first time it is beaten.
include("challenges")

-- West of the start: a curve of rings to take in one sweep. A loop
-- trail round the lot of them does it in one go.
do
    local curve = {}
    for i = 0, 7 do
        local a = i / 8 * 2 * math.pi
        curve[#curve + 1] = { -36 + 9 * math.cos(a), 170 + 9 * math.sin(a) }
    end
    ring_run { name = "zone_ring_run", label = "RING RUN", at = { -20, 170 }, rings = curve,
               seconds = 12, points = 1 }
end

-- From the edge of the start down to the valley floor, 170 m, against the clock.
dash_to { name = "zone_dash", label = "VALLEY DASH", from = { 10, 184 }, to = { 12, 14 },
          seconds = 14, points = 1 }

-- Both sentries in the valley, inside a minute of stepping on the pad.
clear_out { name = "zone_sentries", label = "SENTRIES", at = { 14, 40 },
            enemies = { sentry_west, sentry_east }, seconds = 60, points = 1 }

-- A pad out of the start, pointing down the lane.
dash_pad { at = on_ground(0, 190, 0.1), dir = { 0, 0, -1 }, speed = 26 }
-- And a spring at the foot of the mesa for anyone short of speed.
spring { at = on_ground(64, -140, 0.2), dir = { 0, 1, 0 }, speed = 20 }

-- The goal stands on the mesa, a little under four hundred metres from
-- the start as the crow flies. It is not there until the boss is beaten.
local the_goal = goal(on_ground(70, -170, 2.0))
set_active(the_goal, boss_beaten()) -- this script runs again when a run ends: beaten stays beaten
trigger {
    at = on_ground(70, -170, 0.3),
    radius = 3.0,
    on_touch = function()
        if not active(the_goal) then
            message("the goal is shut: its warden is on the flat ground to the west")
        end
    end,
}

-- The boss. What its fight is like is in boss.lua; here is only where
-- it stands, and the places it runs between when it is chased: once
-- round its own ground, a little under four hundred metres.
local home = { -60, -150 }
local route = { { -20, -120 }, { -60, -95 }, { -120, -105 }, { -155, -150 },
                { -120, -195 }, { -60, -200 }, { -15, -170 } }
boss { at = on_ground(home[1], home[2], 1.0), kind = "warden", radius = 1.0, route = route }

-- From the foot of the mesa to it, and along the way it runs: a boost
-- is paid for in rings, and the chase is all boost.
trail(38, -150, -24, -150, 8)
for i = 1, #route do
    local a, b = route[i], route[i % #route + 1]
    trail(a[1], a[2], b[1], b[2], 10)
end
-- Thirteen metres out from where it stands, the circle to draw round it:
-- outside its pulse, with rings enough to boost most of the way round twice.
-- Its pulses throw out the rest.
for i = 0, 19 do
    local angle = i / 20 * 2 * math.pi
    local p = on_ground(home[1] + 13 * math.cos(angle), home[2] + 13 * math.sin(angle), 1.1)
    ring { p[1], p[2], p[3] }
end

events.on("boss.woke", function()
    message("THE WARDEN")
end)
events.on("boss.defeated", function()
    set_active(the_goal, true)
    local paid = reward("zone_warden", 3)
    message(paid > 0 and "the goal is open: back to the mesa" or "the goal is open")
end)

trigger {
    at = on_ground(0, 194, 0.3),
    radius = 0.9,
    on_touch = function()
        message("the goal is on the mesa to the north-east: follow the rings")
    end,
}
