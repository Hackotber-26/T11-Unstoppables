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

The current home experience is implemented in `pages/home/home.html` and `scripts/core/home.mjs`, with requests centralized in `scripts/core/api.mjs`.

Start the backend from the repository root:

```powershell
docker compose -f infrastructure/docker/docker-compose.yml up --build -d
```

Then run `python -m http.server 8000` from `frontend/` and open `http://localhost:8000`.

The connected flows are rider registration/login, backend fare quote, trip creation/list/read/cancel, pool candidate lookup, wallet history/balance, wallet trip payment, and role-gated admin metrics/simulation/listings/credits. The bearer token is kept in page memory and cleared when the page reloads. Admin accounts must be provisioned outside the public UI.

The current backend does not provide external card processing, self-service wallet funding, SMS/OTP, GPS, driver app assignment actions, trip realtime subscriptions, or verified SafHER matching. The UI does not simulate these as successful operations. Frontend API tests run with `node --test tests/unit/*.test.mjs` under Node 22.
