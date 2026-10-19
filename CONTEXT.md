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
purple crosshatch. Empty positions use dark soil; positive biomass uses sparse
(1–9%), medium (10–39%), rich (40–69%) and dense (70–100%) texture/background
bands. Creature glyphs cover the corresponding foreground texture while its
species-specific background remains visible. The legend documents this overlay
limitation. Rendering is deterministic and uses no simulation randomness.
