-- The kinds of enemy, and what makes each a different fight. Counts are
-- in ticks, sixty to the second. An enemy in a level file with no
-- `kind`, or with one that is not listed here, is the first of these.
-- Edit, save, and press F5 in the game.
--
--   notice          how near you must come before it stops patrolling
--   reach           the distance it likes to strike from
--   patience        ticks it will stand and watch before striking anyway
--   reaction        ticks of your attack's startup it must see before it
--                   can put its guard up: a move quicker than this cannot
--                   be blocked by it
--   guard_ticks     how long the guard stays up
--   punish          true: it strikes the moment you leave yourself open
--                   (a blow in its recovery, or a parry thrown at nothing)
--   move            its blow, from moves.lua

return {
    -- Careful. It watches for a long time, and while it is watching it
    -- blocks even the jab (4 ticks of startup). The way in is while it is
    -- busy: during its own blow's half-second wind-up, or after a parry.
    -- With reaction = 6 the jab beats its guard and three presses kill it.
    { name = "sentry", health = 3, notice = 8, reach = 2.2, approach_speed = 3,
      patience = 45, reaction = 3, guard_ticks = 20, punish = false, move = "enemy_swipe" },

    -- Reckless. It never guards, takes more beating, and goes for every opening.
    { name = "brute", health = 5, notice = 10, reach = 2.2, approach_speed = 2,
      patience = 30, reaction = 999, guard_ticks = 0, punish = true, move = "enemy_swipe" },

    -- The boss of the open zone, in the last phase of its fight (boss.lua
    -- has the rest, and its health: the number here is not used). It sees
    -- you from across its ground, blocks even the jab while it watches,
    -- and goes for every opening. Its blow is slow, wide and its own, and
    -- that blow is the way in: hit it while it winds up and get clear,
    -- turn the blow with a parry, or step back and punish the recovery.
    { name = "warden", health = 6, notice = 60, reach = 2.9, approach_speed = 6,
      patience = 40, reaction = 3, guard_ticks = 24, punish = true, move = "warden_slam" },
}
