# Contributing

Thank you for helping improve NimmaSaarthi.

This repository is designed to support a transit-aware mobility platform with a strong emphasis on simulation, safety, operations, and rider trust. Contributions are welcome from product, engineering, design, and research roles.

---

## Ways to contribute

- improve the product logic or matching behavior
- fix bugs in backend or frontend flows
- add or refine simulation scenarios
- improve documentation and architecture alignment
- strengthen safety, analytics, and observability
- expand datasets and testing coverage

---

## Development workflow

1. Fork or clone the repository.
2. Create a feature branch with a clear name.
3. Keep work scoped to one problem or capability at a time.
4. Validate changes with the relevant tests or local scripts.
5. Submit a pull request with a clear summary and evidence.

---

## Repository layout

```text
backend/      Go services and operational logic
frontend/     web app and experience assets
docs/         product and architecture documentation
datasets/     routes, stations, scenarios, and synthetic data
infrastructure/ container, monitoring, and platform config
spec/         business rules, scenarios, and tariffs
research/     experiments and analytical materials
tools/        migration and scenario tooling
```

---

## Local setup

### Backend

```bash
cd backend
go mod download
```

### Frontend

```bash
cd frontend
python -m http.server 8000
```

Use the local environment that fits your workflow, and keep setup instructions in the PR if the work depends on non-standard tooling.

---

## Code standards

- prefer readable, well-documented Go services
- keep domain boundaries clear across `internal/` modules
- maintain a separation between business logic, transport orchestration, and infrastructure glue
- document assumptions around pricing, matching, safety policy, and dispatch logic
- avoid hidden coupling between simulation code and production service logic

---

## Testing expectations

- run the smallest relevant validation for the change
- add or update tests when fixing logic or introducing new behavior
- prefer integration-level validation for operational flows such as matching, routing, and fare calculation

---

## Pull request guidance

Every PR should include:

- a concise summary of the problem being solved
- the files or areas touched
- any assumptions or constraints
- validation steps or evidence run locally
- screenshots or examples when UX changes are involved

---

## Community expectations

Contributors are expected to be respectful, constructive, and safety-conscious. The project is intended to improve urban mobility outcomes, not to optimize around exploitation, hidden costs, or unsafe behavior.

---

## Questions

Open an issue or start a discussion when a change is unclear or requires cross-domain decision-making.