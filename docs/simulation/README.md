# Simulation Documentation

Simulation is a core operating capability in TrainSync. It is used not just for experimentation, but for validating the product model before live production exposure.

---

## Why simulation matters

Transit-aware mobility decisions are highly sensitive to:

- rider timing and arrival bursts
- driver availability and idle patterns
- bay waiting behavior
- corridor demand complexity
- mixed vehicle types and capacity constraints

All of these variables change quickly, so the system needs a practical way to test decision quality before real-world rollout.

---

## Simulation domains

### Demand simulation
Creates realistic rider flow patterns from stations, corridors, and time-of-day conditions.

### Driver simulation
Replicates availability, idle time, assignment acceptance, and route execution trends.

### Fleet assignment simulation
Evaluates whether vehicles are allocated in a way that improves utilization and reduces empty movement.

### Pooling simulation
Tests whether groups of riders can be matched with acceptable savings and time trade-offs.

### Analytics simulation
Measures wait time, detour cost, occupancy, and economic efficiency across scenarios.

---

## Inputs

- station metadata
- route and corridor maps
- travel times and demand distributions
- driver availability profiles
- fare assumptions and policy settings
- safety and operational constraints

---

## Outputs

The simulation layer should produce:

- pooled trip conversion rates
- rider cost and savings outcomes
- wait-time distributions
- utilization and booked-mile comparisons
- safety and incident coverage metrics
- recommended operational thresholds

---

## Operating principle

Simulation is the product's decision-support engine. It produces the evidence required to make matching, pricing, and geographic roll-out decisions responsibly.
