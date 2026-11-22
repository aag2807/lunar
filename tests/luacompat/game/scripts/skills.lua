-- The skill tree. Three branches, four tiers each.
--
-- This file is data: the game reads it, draws it, and does what it says.
-- A skill needs the one named in `needs`, if any, before it can be
-- bought. What a skill DOES is one or more of three things:
--
--   grants = "drift"            switches on an ability the game has:
--                               air_dash, drift, rail_switch, parry
--   times = { boost_speed = 1.1 }   multiplies a value from tuning.lua
--   plus  = { parry_window = 2 }    adds to one
--   (and a move in moves.lua may say `skill = "cross"`: it cannot be
--   used until that skill is owned)
--
-- A skill that does none of them is refused when the game starts, and
-- so is a name the game does not know. `legacy` marks the moves every
-- player had before there was a tree: a save file from then keeps them.
--
-- `points` and `owned` are what a NEW game begins with.
return {
    points = 2,
    owned = {},
    -- What play is worth. Each enemy pays once in a save file, however
    -- often it is beaten. Finishing a level pays the first time, and
    -- after that only a new best time does.
    points_per_enemy = 1,
    points_per_clear = 3,
    points_per_best = 1,

    branches = { "speed", "combat", "technique" },

    skills = {
        -- Speed
        { id = "air_dash", name = "Air Dash", branch = "speed", tier = 1, cost = 1,
          input = "F in the air", grants = "air_dash", legacy = true,
          text = "Dash once in the air, the way you steer. Brings you up to dash speed, and never slows you." },
        { id = "drift", name = "Drift", branch = "speed", tier = 2, cost = 1, needs = "air_dash",
          input = "Hold Q in a turn", grants = "drift", legacy = true,
          text = "Lean into a corner at full speed and come out of it faster than you went in." },
        { id = "rail_switch", name = "Rail Switch", branch = "speed", tier = 3, cost = 2, needs = "drift",
          input = "Left or Right on a rail", grants = "rail_switch", legacy = true,
          text = "Hop sideways from one rail to the next without losing your grind." },
        { id = "boost_2", name = "Boost II", branch = "speed", tier = 4, cost = 3, needs = "rail_switch",
          input = "Hold E",
          times = { boost_drain = 0.7, boost_speed = 1.1 },
          text = "The boost gauge lasts longer, and the boost drives you to a higher speed." },

        -- Combat
        { id = "cross", name = "Second Strike", branch = "combat", tier = 1, cost = 1,
          input = "Attack, Attack", legacy = true,
          text = "Press Attack again as a jab lands and it turns into a cross." },
        { id = "air_slash", name = "Air Slash", branch = "combat", tier = 2, cost = 1, needs = "cross",
          input = "Attack in the air", legacy = true,
          text = "A wide, quick strike in the air. Quicker out than anything on the ground." },
        { id = "finisher", name = "Finisher", branch = "combat", tier = 3, cost = 2, needs = "air_slash",
          input = "Attack x3", legacy = true,
          text = "A third press ends the chain with a slow blow that hits twice as hard." },
        { id = "heavy_hands", name = "Heavy Hands", branch = "combat", tier = 4, cost = 3, needs = "finisher",
          input = "Always on",
          times = { hit_stun = 1.5 },
          text = "Every blow that lands leaves the enemy reeling for half as long again." },

        -- Technique
        { id = "parry", name = "Parry", branch = "technique", tier = 1, cost = 1,
          input = "K as a blow lands", grants = "parry", legacy = true,
          text = "Turn an attack aside at the last moment and leave its owner open." },
        { id = "quick_guard", name = "Quick Guard", branch = "technique", tier = 2, cost = 2, needs = "parry",
          input = "Always on",
          plus = { parry_cooldown = -8 },
          text = "A parry that misses leaves you open for a shorter time before you can try again." },
        { id = "long_opening", name = "Long Opening", branch = "technique", tier = 3, cost = 2,
          needs = "quick_guard", input = "Always on",
          times = { parry_stun = 1.5 },
          text = "A parried enemy stays open for half as long again: room for the whole chain." },
        { id = "wide_window", name = "Wide Window", branch = "technique", tier = 4, cost = 3,
          needs = "long_opening", input = "Always on",
          plus = { parry_window = 2 },
          text = "The parry stays open two ticks longer. A thirtieth of a second, and you will feel it." },
    },
}
