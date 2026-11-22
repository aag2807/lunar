-- The boss of the open zone: what it is, and how its fight goes.
-- Where it stands and where it runs is the level's business (level3.lua),
-- and so is which kind of opponent it is when it duels (enemies.lua).
-- Counts are in ticks, sixty to the second. Edit, save, and press F5.
--
-- A fight is a list of phases, and each phase can be hurt in one way:
--
--   chase    it runs round its route at `speed`. Come within
--            catch_radius of it WHILE BOOSTING and it loses a hit, then
--            gets away at rest_speed for catch_rest ticks.
--   armour   it goes home at `speed` and stands. Blows do nothing. Every
--            pulse_every ticks a pulse runs out along the ground to
--            pulse_radius, with pulse_warn ticks of warning: be further
--            off, or in the air. A loop of trail closed round it takes
--            a hit.
--   duel     it fights as its kind does, and blows take hits.
--
-- A hit never takes more than its phase has left: nothing, however
-- strong, skips a phase.

return {
    name = "warden",

    wake = 30,          -- metres: it wakes when you come this near
    turn = 90,          -- ticks it takes to get up, and between phases; nothing hurts it then

    catch_radius = 2.6,
    catch_rest   = 100,
    rest_speed   = 32,

    -- duel: the blow that makes this many in a row does not make it
    -- reel, and it has its guard back. With 0 every blow does, and the
    -- first jab to land is the whole fight.
    poise = 2,

    -- The whole fight is boost, and boost is paid for in rings. So that
    -- nobody is left with none and no way to get any: running, it drops
    -- shed_rings every shed_every ticks, and each pulse throws out
    -- pulse_rings, to land where a loop is drawn. They lie about for
    -- ring_life seconds.
    shed_every   = 180,
    shed_rings   = 2,
    pulse_rings  = 5,
    ring_throw   = 8,   -- metres a second: they come to rest 1.6 m out for each, so 13 m
    ring_life    = 10,

    pulse_every  = 150,
    pulse_warn   = 45,
    pulse_radius = 7,

    phases = {
        { kind = "chase",  label = "CATCH IT: BOOST",       health = 3, speed = 22 },
        { kind = "armour", label = "DRAW A LOOP ROUND IT",  health = 2, speed = 14 },
        { kind = "duel",   label = "FIGHT",                 health = 6 },
    },
}
