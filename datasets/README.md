# Datasets

This folder contains versioned inputs for local simulation and validation. Every artifact is classified as `synthetic` or `verified`; synthetic records are fictional and must never be treated as operational facts.

## Layout

- `stations/synthetic_stations.csv`: fictional station nodes in a local meter-based coordinate plane.
- `routes/synthetic_network.csv`: directed, fictional route edges between those nodes.
- `scenarios/*.json`: deterministic operational scenario definitions.
- `synthetic/`: generated demand, driver supply, and safety-event fixtures.
- `verified/`: provenance requirements for externally sourced or field-validated datasets. It intentionally contains no fabricated records.
- `schemas/`: field contracts for source and generated data.
- `scripts/build.py`: standard-library builder and referential-integrity validator.

## Build and validate

From the repository root:

```powershell
python datasets/scripts/build.py build
python datasets/scripts/build.py validate
```

The build is deterministic: the same scenario definitions and source CSVs produce byte-identical generated CSVs. Scenario seeds are explicit and outputs include scenario IDs and classification fields.

## Data policy

No paid data APIs are required. The demo graph uses fictional station IDs and local coordinates, not real latitude/longitude. Do not copy these assets into `verified/`. A verified dataset must include its owner, source URL or internal record, license/permission, retrieval or observation date, geographic/time coverage, transformations, and validation evidence. Sensitive personal or precise trip-location data must not be committed.

Generated outputs are useful for deterministic tests and algorithm development only. They are not calibrated forecasts, routing guidance, fare recommendations, safety assessments, or evidence of real-world performance.
