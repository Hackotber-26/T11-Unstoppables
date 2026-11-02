# Frontend Overview

The frontend is the user-facing experience layer for TrainSync. It is organized around rider, driver, admin, public, and simulation workflows.

---

## Product surfaces

### Rider experience
- trip request flow
- pickup and bay assignment
- live trip monitoring
- payment and rating experience

### Driver experience
- login and availability
- assignment and route delivery
- trip completion and earnings status

### Admin and operations experience
- dashboard metrics
- trip and incident oversight
- user and fleet management
- performance monitoring

### Public experience
- landing pages
- service explanation
- conversion and informational content

---

## Top-level structure

- pages/ — page groups by user type and task
- public/ — public-facing assets and content
- scripts/ — application logic and behavior logic
- styles/ — design tokens, layout, and component styling
- tests/ — automated frontend quality checks
- assets/ — fonts, icons, illustrations, animations, and SVG artwork

---

## Interaction model

The frontend is designed for realtime journey updates and assignment-driven behavior. It should always reflect operational state accurately, because riders and drivers depend on clear and timely information.

---

## Implementation guidance

- keep user journeys aligned with role-based access patterns
- prefer state-driven UI updates over static screen-only assumptions
- keep trip, assignment, and payment states clear and explainable
- support mobile and desktop operational contexts where relevant

## Current API-backed client

`index.html` remains the metro-themed homepage and links to distinct rider, driver, and admin portal routes. `styles/master.css` contains the shared theme tokens and portal components derived from the index palette. The role portals share authentication and navigation infrastructure but render separate role-specific layouts.

Rider and admin pages use API-backed trip, wallet, metric, directory, subscription, and simulation data. Rider transit planning requests Google Routes API itineraries from the backend using the seeded station catalog; the result includes scheduled transit-leg times and provider retrieval time, but is not a guaranteed real-time arrival feed. Driver availability, assigned trips, GPS sharing, document review, earnings estimates, ratings, and station bay workflows use authenticated backend endpoints. SMS/OTP requires Twilio credentials; road navigation requires a configured OSRM service. External card payments, payouts, and transit-authority live feeds are not connected.

Driver road navigation draws the OSRM GeoJSON route on an interactive Leaflet map, and riders see shared live driver locations on the same map. Map tiles come from OpenStreetMap with visible attribution; tile requests disclose the approximate map area viewed to that tile provider. The app validates route geometry and keeps directions available if the map library cannot load.

Migration `000003_namma_metro_station_catalog.sql` seeds the operational Purple, Green, and Yellow line stations and their ordered line stops. The Docker Compose `migrate` service applies this catalog before starting the API.

The current home experience is implemented in `pages/home/home.html` and `scripts/core/home.mjs`, with requests centralized in `scripts/core/api.mjs`.

Start the backend from the repository root:

```powershell
docker compose -f infrastructure/docker/docker-compose.yml up --build -d
```

Then run `python -m http.server 8000` from `frontend/` and open `http://localhost:8000`.

The connected flows are rider registration/login, backend fare quote, trip creation/list/read/cancel, pool candidate lookup, wallet history/balance, wallet trip payment, and role-gated admin metrics/simulation/listings/credits. The bearer token is kept in page memory and cleared when the page reloads. Admin accounts must be provisioned outside the public UI.

Rider, driver, and admin sign-in and registration routes are separate. Rider registration is open; driver and admin accounts require a single-use invitation issued by an administrator. New driver accounts remain unverified and offline until approved. Local demo login details are listed in `backend/README.md`.

Displayed prices and wallet amounts use INR (rupees); amount values sent by the API are represented in paise (integer minor units). Local Compose seeding creates an opening demo wallet credit and sample trip records.

The current backend does not provide external card processing, self-service wallet funding, SMS/OTP, GPS navigation, trip realtime subscriptions, ratings, support tickets, or verified SafHER matching. The UI does not simulate these as successful operations. Frontend API tests run with `node --test tests/unit/*.test.mjs`.
