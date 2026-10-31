# Matching and Pooling

The matching engine is the heart of the TrainSync product. It converts station-side demand into shared journeys that minimize wait time, increase occupancy, and preserve rider trust.

---

## Objective

The engine aims to maximize trip efficiency while keeping individual rider outcomes acceptable.

In practice, this means balancing:

- shared trip compatibility
- trip time and detour length
- destination corridor alignment
- rider wait tolerance
- vehicle capacity and operation cost

---

## Matching inputs

- rider pickup station or bay
- target destination clusters
- time windows around arrival or pickup
- walking preference or allowed detour distance
- driver location and route status
- vehicle capacity and service type

---

## Pool formation strategy

### 1. Group by station and corridor
Riders leaving the same station or nearby transit node are grouped into candidate pools.

### 2. Segment by destination similarity
Trips are matched by similar route direction or destination cluster, not just identical destination strings.

### 3. Evaluate timing compatibility
The system checks whether riders can be served without excessive waiting or route delay.

### 4. Filter by safety and rider preference
A rider's safety, comfort, or walking tolerance can alter the accepted match set.

### 5. Assign capacity and vehicle type
The pool is matched to the vehicle best suited for its size, route, and operating conditions.

---

## Matching heuristics

A practical implementation often includes a score function like:

- positive weight for corridor alignment
- positive weight for rider savings
- penalty for detour length
- penalty for pickup delay
- penalty for route complexity
- penalty for safety or trust mismatches

This score should prioritize efficient pooled outcomes without degrading the rider experience unfairly.

---

## Acceptance rules

A pool should only form if it meets minimum conditions such as:

- rider wait times remain reasonable
- detour limits are not exceeded
- total occupancy improves compared with independent rides
- safety policies remain satisfied
- expected fare savings remain visible and meaningful

---

## Operational considerations

Pooling is more effective when the platform knows:

- which riders are likely to board in the same station window
- what the corridor demand pattern looks like
- how long riders are willing to walk before pickup
- which vehicle classes are available at each station node

The engine should continually update its assumptions using trip outcomes and analytics.
