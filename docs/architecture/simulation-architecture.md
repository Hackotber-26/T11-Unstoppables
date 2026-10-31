# Simulation Architecture

## Purpose

The simulation layer is central to NimmaSaarthi's ability to validate the platform before scaling operations. It converts mobility demand assumptions and route dynamics into measurable outcomes.

---

## Simulation inputs

- station and metro flow data
- route and corridor definitions
- rider demand patterns
- driver availability and activity profiles
- fleet capacity and operating constraints
- scenario configurations and policy assumptions

---

## Simulation responsibilities

### Demand generation
Models rider emergence based on station and corridor patterns.

### Driver behavior
Simulates driver readiness, assignment acceptance, and utilization patterns.

### Pooling logic
Tests whether trip clustering produces higher efficiency than isolated rides.

### Routing and bay assignment
Measures how route sequencing and pickup planning affect wait time and trip completion.

### Analytics
Aggregates outcome metrics such as occupancy, wait times, trip savings, and revenue efficiency.

---

## Why simulation first

The product depends on complex operational decisions made under time pressure. Simulation is a safer and cheaper way to test decisions around matching, pricing, route assignment, and fleet allocation before exposing them to real-world users.

---

## Typical decision outcomes

- whether a corridor should be pooled
- how much wait time is acceptable before rerouting
- the right balance between fare affordability and driver earnings
- which pickup zone strategy minimizes empty travel
- how operational policies affect rider trust and platform efficiency

---

## System impact

Simulation is not just a research tool. It is a control layer that helps the platform learn and evolve as demand patterns change.