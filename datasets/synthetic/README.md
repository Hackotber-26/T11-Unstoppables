# Generated Synthetic Fixtures

Run `python datasets/scripts/build.py build` from the repository root to regenerate the CSVs. The generator uses only Python's standard library and the seeds in `scenarios/*.json`.

- `demand_requests.csv`: fictional rider requests and constraints.
- `driver_supply.csv`: fictional driver/vehicle availability for each scenario.
- `safety_events.csv`: fictional review events linked to generated requests.

The `available` and `accessible_vehicle` fields are scenario inputs, not verified claims about real people or vehicles. Outputs are replaceable build artifacts; edit scenario definitions or source files, then regenerate instead of hand-editing generated CSVs.