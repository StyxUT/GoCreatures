# Vegetation model (foundation slice)

Each map cell has one shared biomass capacity of 100 units and three fixed
species slots, ordered grass, clover, bramble. A species slot with positive
biomass is a local colony; zero biomass means no colony and does not regrow
spontaneously. `PlantCellAt` returns a detached `PlantCellState` value snapshot
with the shared capacity and, for each species, identity, presence, biomass,
generation, growth progress, regrowth rate, spread probability and nutritional
yield. The snapshot is the stable read contract for renderers and later
propagation work; mutating it does not change simulation state.

| Species | Regrowth per tick | Spread probability | Spread threshold | Nutrition per biomass |
| --- | ---: | ---: | ---: | ---: |
| Grass | 0.08 | 0.02 | 20 | 15 |
| Clover | 0.04 | 0.01 | 30 | 20 |
| Bramble | 0.02 | 0.005 | 40 | 10 |

Initial maps seed `6 * max(1, ceil(cellCount/300))` center cells per species. A
center receives 20 biomass and its in-bounds cardinal halo cells receive 5.
Coincident patches are proportionally scaled if their total would exceed shared
capacity. Thus a one-cell map still gets all three species and remains below
capacity; empty maps contain no cells. Plants grow only where a species already
has a local colony. Its fractional growth is its species rate multiplied by the
cell's free-capacity fraction. Competing growth is allocated proportionally,
not by a fixed species priority, and biomass remains fractional. Grass attempts
cardinal spread, clover targets any of eight adjacent cells, and bramble targets
cells within Chebyshev distance two. A candidate colony must meet the species
threshold; a successful establishment transfers one biomass from its parent.
Targets reject an existing same-species colony and cells without one free
biomass unit. All attempts use a tick-start snapshot, so new colonies cannot
spread during their establishment tick.

## Local cohorts and evolution

Each occupied species slot is one local cohort with a generation and inherited
regrowth, spread-probability and nutritional-yield traits. Founder colonies use
the baseline table above and generation zero. A successful establishment copies
the parent's cohort traits and increments generation. It then has a 0.1% chance
to mutate exactly one of those traits by a uniform relative change in
[-0.25%, +0.25%], clamped to 80–120% of the species baseline. Traits remain
fractional and local: regrowth uses the cohort rate, spreading uses its
probability, and herbivore intake and seeking use its nutritional yield.
Ordinary regrowth, failed attempts and occupied-target rejection do not mutate.

When grazing or propagation reduces a local colony to zero, its cohort and
generation are discarded. Recolonization creates a new daughter from the
successful parent, not from stale state at the empty target. One cohort per
species per cell is a deliberate simplification; species identity and renderer
texture never mutate.

In a deterministic sample of 100,000 grass inheritance draws (seed 20261019),
101 cohorts mutated (0.101%). These draws exercise the same inheritance helper
called after successful establishment. By comparison, creature cloning
perturbs five continuous traits by about ±1% on each clone. Plant mutations are
therefore much rarer per establishment and individually smaller; this documents
the configured event frequency, not an expected wall-clock evolution time.

Herbivores consume at most five biomass units in one action, selecting the
highest-yield species first. Equal yields follow the fixed species order.
Nutrition is capped by the creature's maximum; fractional nutrition is retained
between actions, and tick statistics record actual biomass consumed (including
fractional amounts). Carnivores do not consume plants.

Food seeking ranks observed cells by the same total nutritional value used by
eating: `sum(biomass * nutritional yield)`. The previous movement threshold of
30 grass-equivalent biomass units converts to 450 nutritional-value units
(`30 * 15`). Tick reports expose per-species biomass, total shared capacity,
biomass added by regrowth and biomass consumed, all sourced from the plant
populations rather than a separately mutable food cache.
