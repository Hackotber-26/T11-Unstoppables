# Routing and ETA

Routing and ETA logic sit at the intersection of trip planning, live operations, and rider trust. They determine where the vehicle should be, how it should get there, and when the rider should expect the pickup.

---

## Purpose

The routing engine optimizes the journey from station or bay to final destination while preserving pool quality and minimizing empty travel. The ETA engine translates network conditions into reliable time estimates for riders and drivers.

---

## Routing inputs

- current vehicle position
- station or pickup location
- route network and segment speed estimates
- active rider drop sequence
- traffic and congestion conditions
- service policy constraints
- bay assignment and walk-distance rules

---

## Route optimization goals

- minimize deadhead and empty mileage
- preserve rider comfort and trip feasibility
- respect pickup windows and drop-order priorities
- support pooled trip sequencing with multiple riders
- maintain acceptable driver utilization

---

## ETA estimation model

A practical ETA model should combine:

- historical corridor travel time
- live network conditions
- queue and station dwell time
- pickup and bay walking time
- traffic variance and weather-based effects when relevant

This makes predictive ETAs more accurate than pure static route estimates.

---

## Drop sequencing

When multiple riders are included in a shared trip, sequence decisions should consider:

- nearest drop-off ordering
- corridor alignment
- rider waiting tolerance
- pickup-to-destination travel efficiency
- trip fairness and stop burden

The system must prevent a route from becoming overly complex or unbalanced just to satisfy one rider.

---

## Operational impact

Good routing and ETA logic improves:

- rider confidence in the service
- driver efficiency and utilization
- plan stability under real-time changes
- bay assignment quality
- analytics for future demand tuning

---

## Design guidance

A robust implementation should make routing decisions explainable to operations teams and should expose the key variables feeding the ETA model. This matters both for trust and for debugging when service quality degrades.
