# Data Architecture

The platform stores both transactional operational data and analytical intelligence data.

---

## Core storage systems

### PostgreSQL
Used for the canonical data model for users, trips, drivers, stations, vehicles, fares, incidents, subscriptions, and wallet state.

### Redis
Used for ephemeral coordination, cache lookups, live event state, and short-lived operational data.

### Datasets and scenarios
The `datasets/` directory stores station data, synthetic transit patterns, and scenario definitions that support simulation and validation.

---

## Core data domains

- users and profiles
- riders and drivers
- stations and metro nodes
- vehicles and assignments
- trip lifecycle and status events
- routing and ETA history
- pricing and fare output
- wallet balances and settlement records
- subscriptions and recurring plans
- incidents and safety review
- simulation inputs and generated outputs

---

## Data flow pattern

1. event enters the system from rider, driver, or station data
2. the relevant service validates the event and stores transactional state
3. related aggregates are updated for matching, pricing, and dashboards
4. realtime notifications are emitted to client sessions
5. analytics and reporting queries use both live and historical data

---

## Design principles

- keep authoritative datasets in transactional stores
- treat realtime state as derived and refreshable
- separate operational state from long-term analytics storage patterns
- maintain a clear relationship between demand events and trip outcomes
- retain safety and audit data for investigation and accountability