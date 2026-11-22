-- Particles: what each kind of burst looks like, and which of the game's
-- events sets one off.
--
-- A kind: how many (`count`), how fast (`speed`, units a second), how
-- widely thrown (`spread`: 0 is one way, 1 is every way), how long each
-- lives (`life`, seconds), how big (`size`: at birth, at death), what
-- color (`color`, and `color_end` if it changes; above 1 is brighter than
-- white), how hard it is pulled down (`gravity`; negative floats up) and
-- how quickly it slows (`drag`). `stretch` and `scatter` are explained
-- at the one kind that uses them. A pair { a, b } means "somewhere from a
-- to b", chosen for each particle.
--
-- A rule: when `event` is posted, make a burst of `kind` where the event
-- happened, `lift` above it, thrown `along` "up", "back" (behind the
-- player) or "forward".
--
-- A trail: see the bottom of the file.
--
-- They are light added to the picture, so they show on dark things and
-- vanish against a bright sky. Edit, save, and press F5 in the game.
return {
    kinds = {
        { name = "spark", count = 16, speed = { 3, 8 }, spread = 0.9, life = { 0.15, 0.35 },
          size = { 0.16, 0.03 }, color = { 1.6, 1.3, 0.6 }, color_end = { 1.2, 0.3, 0.05 },
          gravity = 9, drag = 2 },
        { name = "clank", count = 12, speed = { 3, 7 }, spread = 0.7, life = { 0.15, 0.3 },
          size = { 0.16, 0.03 }, color = { 0.8, 1.1, 1.6 }, gravity = 4, drag = 3 },
        { name = "parry", count = 28, speed = { 5, 11 }, spread = 1.0, life = { 0.25, 0.5 },
          size = { 0.22, 0.02 }, color = { 0.6, 1.4, 1.8 }, color_end = { 0.2, 0.5, 1.6 },
          gravity = 2, drag = 4 },
        { name = "burst", count = 40, speed = { 2, 9 }, spread = 1.0, life = { 0.4, 0.9 },
          size = { 0.28, 0.0 }, color = { 1.6, 0.5, 0.3 }, color_end = { 0.8, 0.1, 0.05 },
          gravity = 6, drag = 1.5 },
        { name = "streak", count = 22, speed = { 3, 9 }, spread = 0.45, life = { 0.25, 0.5 },
          size = { 0.3, 0.05 }, color = { 0.5, 1.0, 1.6 }, drag = 3 },
        { name = "dust", count = 30, speed = { 3, 7 }, spread = 0.95, life = { 0.3, 0.7 },
          size = { 0.25, 0.7 }, color = { 0.8, 0.75, 0.6 }, gravity = -1, drag = 4 },
        { name = "hop", count = 8, speed = { 1, 3 }, spread = 0.7, life = { 0.15, 0.3 },
          size = { 0.12, 0.02 }, color = { 1.4, 1.4, 1.0 }, gravity = 5, drag = 2 },
        { name = "flare", count = 20, speed = { 3, 7 }, spread = 0.35, life = { 0.2, 0.45 },
          size = { 0.26, 0.04 }, color = { 0.4, 0.9, 1.8 }, drag = 3 },
        { name = "glitter", count = 18, speed = { 1, 3 }, spread = 0.5, life = { 0.5, 1.0 },
          size = { 0.1, 0.02 }, color = { 1.8, 1.5, 0.4 }, gravity = -2, drag = 1 },
        -- Small ones, made over and over while something goes on.
        { name = "jet", count = 3, speed = { 2, 5 }, spread = 0.3, life = { 0.2, 0.4 },
          size = { 0.24, 0.03 }, color = { 0.4, 0.9, 1.8 }, drag = 2 },
        { name = "grit", count = 2, speed = { 1, 3 }, spread = 0.8, life = { 0.2, 0.4 },
          size = { 0.1, 0.25 }, color = { 0.7, 0.65, 0.5 }, gravity = -1, drag = 3 },
        { name = "glint", count = 2, speed = { 2, 5 }, spread = 0.7, life = { 0.15, 0.3 },
          size = { 0.09, 0.02 }, color = { 1.6, 1.4, 0.8 }, gravity = 8, drag = 1 },
        -- Not a dot but a line: `stretch` draws each as the distance it
        -- covers in that many seconds, and `scatter` starts each up to
        -- that many metres from the burst. The air, at speed.
        { name = "shock", count = 48, speed = { 9, 11 }, spread = 1.0, life = { 0.5, 0.65 },
          size = { 0.3, 0.05 }, color = { 1.8, 0.5, 0.3 }, color_end = { 0.9, 0.1, 0.05 }, drag = 0.2 },
        { name = "wind", count = 3, speed = { 16, 24 }, spread = 0.04, life = { 0.2, 0.35 },
          size = { 0.05, 0.03 }, color = { 1.2, 1.4, 1.8 }, stretch = 0.1, scatter = 4.0 },
    },

    on = {
        { event = "enemy.hit",      kind = "spark" },
        { event = "shard.collected", kind = "glitter", lift = 0.4 },
        { event = "loop.closed",    kind = "flare", lift = 0.6 },
        { event = "enemy.blocked",  kind = "clank" },
        { event = "parry.landed",   kind = "parry" },
        { event = "enemy.defeated", kind = "burst" },
        { event = "air.dashed",     kind = "streak",  lift = 1.0, along = "back" },
        { event = "boost.started",  kind = "flare",   lift = 0.8, along = "back" },
        { event = "stomp.landed",   kind = "dust",    lift = 0.1 },
        { event = "rail.switched",  kind = "hop",     lift = 0.1 },
        { event = "points.earned",  kind = "glitter", lift = 1.2 },
        { event = "boss.woke",      kind = "dust",    lift = 0.1 },
        { event = "boss.hurt",      kind = "parry",   lift = 0.4 },
        { event = "boss.phase",     kind = "burst",   lift = 0.6 },
        { event = "boss.pulse",     kind = "shock",   lift = -0.8 },
        { event = "boss.defeated",  kind = "burst",   lift = 1.0 },
        { event = "boss.defeated",  kind = "glitter", lift = 1.6 },
    },

    -- A trail: while the player is `during` something ("boosting",
    -- "drifting", "rolling", "grinding" or "speeding"), a burst of `kind` every
    -- `every` seconds, where the player's feet are.
    trails = {
        { during = "boosting", kind = "jet",   every = 0.03, lift = 0.8, along = "back" },
        { during = "drifting", kind = "grit",  every = 0.04, lift = 0.1 },
        { during = "rolling",  kind = "grit",  every = 0.06, lift = 0.1 },
        { during = "grinding", kind = "glint", every = 0.04, lift = 0.05 },
        { during = "speeding", kind = "wind",  every = 0.02, lift = 1.4, along = "back" },
    },
}
