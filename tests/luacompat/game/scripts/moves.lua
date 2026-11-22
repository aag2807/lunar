-- The player's attacks. Everything is counted in ticks, sixty to the
-- second: `startup` ticks before anything can be hit, `active` ticks
-- with the hitbox out, `recovery` ticks before the attacker is free.
-- The hitbox is a ball `reach` ahead of the feet, `height` above them,
-- of this `radius`. A hit holds the world still for `hit_stop` ticks. Pressing attack again turns a move into `cancel_to`,
-- from tick `cancel_from` on. A move with `skill` cannot be used until
-- that skill from skills.lua is owned. Edit, save, and press F5 in the game.
--
-- This file can do nothing but return a table: no os, no io, no print.

return {
    -- The ground chain: jab, cross, and a slower blow to finish.
    { name = "jab",   startup = 4, active = 3, recovery = 10,
      reach = 1.2, height = 0.9, radius = 0.9, damage = 1, hit_stop = 3,
      cancel_to = "cross", cancel_from = 7 },
    { name = "cross", skill = "cross", startup = 5, active = 3, recovery = 12,
      reach = 1.3, height = 0.9, radius = 0.9, damage = 1, hit_stop = 4,
      cancel_to = "finisher", cancel_from = 8 },
    { name = "finisher", skill = "finisher", startup = 8, active = 4, recovery = 20,
      reach = 1.5, height = 0.9, radius = 1.1, damage = 2, hit_stop = 6 },

    -- What an enemy does to you. Half a second of warning is the whole
    -- point: it is there to be read, and parried.
    { name = "enemy_swipe", enemy = true, startup = 30, active = 3, recovery = 30,
      reach = 1.2, radius = 1.0, damage = 1, hit_stop = 4 },

    -- The boss's: longer to come, wider, and longer to get over.
    { name = "warden_slam", enemy = true, startup = 36, active = 4, recovery = 40,
      reach = 1.9, radius = 1.6, damage = 1, hit_stop = 6 },

    -- In the air: quick to come out, wide, and nothing after it.
    { name = "air_slash", skill = "air_slash", air = true, startup = 3, active = 5, recovery = 8,
      reach = 1.0, height = 0.6, radius = 1.1, damage = 1, hit_stop = 3 },
}
