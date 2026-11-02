# ADR 0001: Simulation-First Architecture

- Status: Accepted
- Date: 2026-10-31

---

## Context

TrainSync is a transit-aware mobility platform with a high degree of operational complexity. A generic ride-hailing model would not be sufficient because the system depends on station timing, rider pooling, driver positioning, and corridor-level route behavior.

At the same time, live experiments on real riders and drivers are costly and risky. Many high-impact decisions—matching policy, pickup zoning, and fare logic—must be tested before being exposed to production traffic.

---

## Decision

We will treat simulation as the primary validation mechanism for matching, routing, pooling, and operational policy decisions.

This means:

- demand patterns are modeled before feature rollout
- trip simulations are run to evaluate operational performance
- matching heuristics are tuned using scenario-based metrics
- deployment decisions incorporate simulation evidence rather than intuition alone

---

## Consequences

### Positive
- lower risk before live operations
- faster iteration on corridor and route decisions
- stronger evidence for safety and pricing policy changes
- clearer analytics for operational optimization

### Negative
- simulation quality depends on good scenario design and data quality
- over-optimization in simulation can create mismatches in the real world
- infrastructure and tooling for simulation must be maintained as a first-class system component

---

## Rationale

The line between a useful mobility platform and a chaotic one is the accuracy of the operational assumptions. Simulation gives the team a controlled environment to test those assumptions at scale.

This approach also aligns with the repository structure, which includes datasets, simulation tools, and operational models under the backend and research layers.
