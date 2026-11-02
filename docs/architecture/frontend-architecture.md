# Frontend Architecture

The frontend is designed to support multiple user types and operational surfaces through a shared web experience layer.

---

## Stack

- HTML for structure
- CSS for styling and layouts
- JavaScript for interaction and workflow logic
- Canvas and SVG for maps, trip paths, and station visuals
- Web Workers for background processing and UI offloading

---

## Experience groups

### Rider flows
- login and onboarding
- request pooling and trip state
- pickup and bay assignment
- live trip events
- payments and ratings

### Driver flows
- login and availability status
- assignment and route guidance
- pickup confirmation
- trip completion and earnings view

### Admin and operations flows
- dashboard summaries
- rider and driver management
- trip and incident monitoring
- analytics and reporting

### Public flows
- landing pages, service explanations, and conversion-oriented interfaces

---

## Page organization

The project separates page groups by audience:

- `frontend/pages/admin/`
- `frontend/pages/auth/`
- `frontend/pages/driver/`
- `frontend/pages/rider/`
- `frontend/pages/public/`
- `frontend/pages/simulation/`

This keeps the user journeys aligned with user roles and operational concerns.

---

## Interaction model

The frontend depends on live data updates and event-driven states rather than static page transitions. It is designed to represent trip updates, route changes, pool status, and assignment transitions in a way that feels immediate and operationally trustworthy.

---

## UI goals

- low-friction rider experience
- clear assignment and pickup states
- dependable operational visibility for staff
- strong use of maps, annotations, and flow-oriented interfaces
- responsiveness across travel and dispatch patterns