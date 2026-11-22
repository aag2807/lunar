-- Stats: what the shards hidden in the open zone are worth.
--
-- This file is data. Each shard found raises one stat by one level, up
-- to that stat's `max`, and every level changes values from tuning.lua:
--
--   times = { top_speed = 1.03 }   multiplies it, once for each level
--   plus  = { parry_window = 1 }   adds to it, once for each level
--
-- A value the tuning does not have is refused when the game starts.
-- The zone has three shards for each stat; where they are is in
-- level3.lua.
return {
    stats = {
        { id = "speed", name = "Speed", max = 3,
          times = { top_speed = 1.03, sprint_speed = 1.03, boost_speed = 1.02 },
          text = "Runs, sprints and boosts a little faster." },
        { id = "force", name = "Force", max = 3,
          times = { hit_stun = 1.15, parry_stun = 1.1 },
          text = "Enemies reel for longer when struck or parried." },
        { id = "stamina", name = "Stamina", max = 3,
          times = { boost_drain = 0.9 },
          text = "The boost gauge empties more slowly." },
    },
}
