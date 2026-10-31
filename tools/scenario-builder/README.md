# Scenario Builder

`build.py` derives a new synthetic scenario from an existing validated template and writes it under `datasets/scenarios/`. It requires an explicit ID and seed and refuses to overwrite an existing scenario without `--force`.

The generated scenario is checked with the same constraints as `datasets/scripts/build.py`. It does not create real demand observations.