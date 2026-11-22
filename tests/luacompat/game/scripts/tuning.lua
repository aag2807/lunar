-- How the character feels. Edit a number, save, and press F5 in the game.
-- Anything left out keeps the value built into the C code.

local gravity = 30

return {
    gravity             = gravity,
    top_speed           = 14,
    acceleration        = 24,
    sprint_speed        = 30,
    sprint_acceleration = 40,
    friction            = 10,
    drag                = 0.010,

    turn_rate           = 6,
    turn_ease_speed     = 12,
    turn_loss           = 0.35,

    -- A jump that reaches 2 units: v = sqrt(2 * g * h).
    jump_speed          = math.sqrt(2 * gravity * 2.0),
    jump_cut            = 0.45,
    air_acceleration    = 12,

    cling_speed         = 7,
    wall_jump_push      = 9,
    wall_run_time       = 0.9,
    wall_run_gravity    = 0.25,
    wall_run_speed      = 7,

    -- Rolling: hold Left Ctrl. A ball has no legs, so the ground takes
    -- little from it and a slope gives it more (and costs it less).
    roll_min_speed      = 4,
    roll_stop_speed     = 2,
    roll_friction       = 1.5,
    roll_drag           = 0.004,
    roll_downhill       = 1.6,
    roll_uphill         = 0.6,
    roll_turn           = 0.4,

    -- The boost gauge: hold E. 0.25 a second empties a full gauge in four.
    boost_speed         = 38,
    boost_acceleration  = 70,
    boost_drain         = 0.25,
    boost_min           = 0.1,
    boost_per_ring      = 0.08,

    -- Drifting: hold Q in a turn. The charge is per radian turned, so
    -- 0.7 fills it in a little over 80 degrees.
    drift_min_speed     = 10,
    drift_turn          = 2.4,
    drift_loss          = 0.05,
    drift_charge        = 0.7,
    drift_boost         = 8,
    drift_gauge         = 0.15,

    -- In the air: F is one dash for each trip through the air, and
    -- Left Ctrl, which rolls on the ground, stomps straight down.
    air_dash_speed      = 26,
    air_dash_time       = 0.18,
    stomp_speed         = 45,
    stomp_keep          = 0.15,

    -- On a rail, lean left or right to hop to a rail beside it.
    rail_switch_reach   = 4,
    rail_switch_time    = 0.15,
    rail_switch_hop     = 0.5,

    -- The camera: 60 degrees standing, 78 at boost speed, and 2 units
    -- further back. Set lens_wide to 60 and lens_pull to 0 to switch it off.
    lens_rest           = 60,
    lens_wide           = 78,
    lens_slow_speed     = 14,
    lens_fast_speed     = 38,
    lens_rate           = 4,
    lens_pull           = 2,

    -- The duel. R locks on to the nearest enemy ahead. K parries: one
    -- press is good for parry_window ticks (sixty to the second), then
    -- the button is dead for parry_cooldown more.
    parry_window        = 6,
    parry_cooldown      = 24,
    parry_stun          = 1.2,
    parry_freeze        = 8,
    lock_range          = 12,
    -- An enemy with the player this near waits this many ticks, then strikes.
    enemy_strike_range  = 3,
    enemy_strike_wait   = 45,
    duel_camera_min     = 6,
    duel_camera_max     = 18,
    duel_camera_margin  = 1.5,
    duel_camera_side    = 35,  -- degrees: straight behind (0), the player hides the enemy

    -- The loop trail: boost along the ground and a line of light follows.
    -- Cross it before it fades and everything inside the loop is yours.
    loop_spacing        = 0.5, -- metres between the trail's points
    loop_life           = 4,   -- seconds a point lasts: at boost speed, a loop of about 120 m
    loop_min_area       = 4,   -- square metres: smaller than this is a wobble, not a loop
    loop_damage         = 1,   -- hits an enemy inside takes
}
