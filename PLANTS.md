# Vegetation model (foundation slice)

Each map cell has one shared biomass capacity of 100 units and three fixed
species slots, ordered grass, clover, bramble. A species slot with positive
biomass is a local colony; zero biomass means no colony and does not regrow
spontaneously. `PlantCellAt` returns a detached `PlantCellState` value snapshot
with the shared capacity and, for each species, identity, presence, biomass,
growth progress, regrowth rate and nutritional yield. The snapshot is the stable
read contract for renderers and later propagation work; mutating it does not
change simulation state.

| Species | Regrowth per tick | Nutrition per biomass unit |
| --- | ---: | ---: |
| Grass | 0.08 | 15 |
| Clover | 0.04 | 20 |
| Bramble | 0.02 | 10 |

Initial maps seed `max(1, ceil(cellCount/300))` center cells per species. A
center receives 20 biomass and its in-bounds cardinal halo cells receive 5.
Coincident patches are proportionally scaled if their total would exceed shared
capacity. Thus a one-cell map still gets all three species and remains below
capacity; empty maps contain no cells. Plants grow only where a species already
has a local colony. Its fractional growth is its species rate multiplied by the
cell's free-capacity fraction. Competing growth is allocated proportionally,
not by a fixed species priority, and biomass remains fractional. Propagation and
trait evolution are left to their dependent slices.

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
