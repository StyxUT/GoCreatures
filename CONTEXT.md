# GoCreatures

A tick-based creature simulation for experimenting with diet, inheritance and
competition. These terms describe the simulation's state and accounting.

## Language

**Creature**:
An individual with an identity, inherited traits, a diet and mutable health,
nutrition and position. Positive health makes it eligible to act.

**Ancestor**:
A creature created without a parent, and the root identity shared by its lineage.

**Clone**:
A descendant whose traits are derived from its parent's pre-cloning state;
cloning then charges the parent and temporarily immobilizes both creatures.

**Diet**:
Herbivore or Carnivore; determines ancestor stat ranges and action policy, and is
inherited by clones.

**Tick**:
An ordered simulation step that maintains the environment, updates creatures,
collects decisions, resolves actions and completes statistics.

**Action intent**:
A creature's decision collected before actions are resolved. An intent can be
suppressed when combat or death makes its actor ineligible.

**Tick statistics**:
Events recorded during one tick plus a final snapshot of surviving creatures;
population and age, health and nutrition totals all describe that same snapshot.

**Death**:
A live creature's removal for age at tick entry, or a live-to-dead health
transition from starvation or fighting; each death is credited once.

**Carnivore totals**:
Cumulative action and death counts for carnivores across ticks, separate from
the final population snapshot in tick statistics.

## Vegetation display

Each map cell occupies three terminal columns in stable grass, clover, bramble
order. Grass uses olive vertical strokes, clover teal stipple, and bramble muted
purple crosshatch. Empty positions use plain dark soil with no foreground
texture; positive biomass uses sparse
(1–9%), medium (10–39%), rich (40–69%) and dense (70–100%) texture/background
bands. Creature glyphs cover the corresponding foreground texture while its
species-specific background remains visible. The legend documents this overlay
limitation. Single carnivores use brackets; multiple carnivores and crowded
herbivores retain a compact count cue within the three-column cell. Rendering is
deterministic and uses no simulation randomness.

## Starting creatures and predator avoidance

Each cell has a 4.5% initial creature spawn probability (half the previous 9%).
Of those spawns, 5% are carnivores and 95% herbivores, halving the expected
starting population of each diet. Manual spawns still create one creature.

A mobile herbivore prioritizes escape over fighting, cloning, feeding and
resting when a neighboring step increases its distance from the nearest known
living carnivore. Known threats come from the current cell (even at zero vision)
and the cells in its normal observations; unseen or dead predators do not affect
the decision. All eight in-bounds neighboring steps are considered, using
Chebyshev distance to match diagonal movement and adjacent predator attacks.
Equal-best steps use stable creature-identity tie-breaking, not random draws or
unseen cell contents. Immobile herbivores keep their normal non-movement actions.

If no strictly safer step exists, normal actions remain available, but movement
closer to a known predator or into its cell becomes resting instead. Escape is
an ordinary move intent: existing fight-before-movement resolution still applies,
so choosing to flee does not guarantee evading an attack already in range.

## Plant ecology

Each cell has a shared capacity of 100 biomass units. Founding patches use up to
`2 * max(1, ceil(cellCount/300))` centers per species, with 20 biomass at a center
and 5 in each in-bounds cardinal halo cell; overlapping requests are scaled to
fit capacity. Each tick snapshots vegetation, accumulates fractional growth
from existing colonies at grass/clover/bramble rates 0.08/0.04/0.02 scaled by
the free-capacity fraction, then resolves local spread intents. Eligible
colonies (20/30/40 biomass) attempt spread with probability 0.02/0.01/0.005
respectively, targeting one uniformly selected cardinal/four-neighbor/eight
neighbor-within-distance-two offset. Establishment requires a free species slot,
one biomass unit of source and destination capacity; that unit transfers from
parent to daughter colony. Tick-start snapshots prevent same-tick cascades.
Growth (`BiomassAdded`), establishment transfers (`BiomassTransferred`) and
grazing (`BiomassConsumed`) are accounted separately. Empty regions and extinct
species do not spontaneously reappear.

## Plant settings and reports

| Species | Regrowth / tick | Nutrition / biomass | Spread threshold | Attempt / eligible tick | Neighborhood |
| --- | ---: | ---: | ---: | ---: | --- |
| Grass | 0.08 | 15 | 20 | 0.02 | cardinal |
| Clover | 0.04 | 20 | 30 | 0.01 | eight adjacent cells |
| Bramble | 0.02 | 10 | 40 | 0.005 | within Chebyshev distance two |

Each successful establishment copies the parent's local cohort and has a 0.001
mutation probability. At most one of regrowth, spread probability or nutrition
changes by up to 0.25% relative; values stay within 80–120% of the species
baseline. The report shows final-snapshot biomass, occupied cells and
biomass-weighted trait means, plus per-tick establishment/mutation/grazing and
cumulative growth, transfers, consumption and evolution events. A species with
no living colonies reports zero biomass, occupancy and trait means. The seeded
diagnostic is `go test -run '^TestSeededPlantMutationRateIsRareRelativeToCreatureCloning$' -count=1 -v`; it samples 100,000 establishment opportunities and records realized plant mutations against the larger per-clone creature trait perturbations. This is an opportunity-rate comparison, not a wall-clock coexistence guarantee.

## Integration performance evidence

Measured on Go 1.27.2, linux/amd64, AMD Ryzen 9 3950X with
`go test -run '^$' -bench '^BenchmarkDefaultWorld(TickAndRender|Tick|Render|PlantMaintenance)$' -benchtime=100x -count=5 -benchmem`.
The comparison revision is pre-vegetation `b497f8b`; its benchmark adapter
changes only the historical `CreatureFactory` and `simulateTick` signatures.
Reported values are medians of five runs:

| Scenario | Pre-vegetation | Integrated plants | Change |
| --- | ---: | ---: | ---: |
| 38x81 tick + map render | 285 µs, 224 KB | 565 µs, 461 KB | 1.99x time |
| 38x81 tick only | 94 µs, 7.7 KB | 311 µs, 83.6 KB | 3.30x time |
| 38x81 map render only | 159 µs, 193 KB | 184 µs, 378 KB | 1.16x time |
| 38x81 plant maintenance, no creatures | 59 µs, 128 B | 135 µs, 48 B | +76 µs |

Profiling identified a full per-tick grid copy for plant snapshots and oversized
render-buffer allocation. Spread intents are now collected before any plant
mutation (preserving snapshot semantics without copying the grid), and render
output is pre-sized with cached species styles. This reduced the combined
integrated benchmark from about 956 µs / 1.38 MB per operation to the recorded
565 µs / 461 KB. The remaining absolute tick+render cost is under 0.6 ms versus
the normal 300 ms tick interval; the ecology parameters were retained because
the seeded checks showed the requested bounded, rare mutation rate and no
resource invariant failures.
