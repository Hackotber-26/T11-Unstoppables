# Tools Overview

The tools directory contains supporting utilities used to build, test, migrate, and inspect the TrainSync platform.

---

## Included toolchains

- benchmark/ — performance measurement and comparison utilities
- migration-tools/ — database and schema migration utilities
- scenario-builder/ — scenario generation and operational setup tooling
- vector-generator/ — synthetic vector and movement generation utilities

---

## Purpose

These tools support the platform’s core engineering needs: faster scenario testing, better migration safety, improved data generation, and realistic system evaluation.

---

## Guidance

Tooling should be kept small, reproducible, and easy to use. When a tool changes the behavior of the system, its assumptions and outputs should be documented as part of the engineering workflow.

## Commands

All tools use Python's standard library; no paid API or third-party package is required. Run commands from the repository root:

```powershell
python tools/scenario-builder/build.py --id afternoon_peak --seed 73001 --riders 160
python tools/vector-generator/generate.py --output tools/vector-generator/generated/synthetic-network.json --path ST-A1 ST-A4
python tools/vector-generator/generate.py --output tools/vector-generator/generated/synthetic-network.json --path ST-A1 ST-A4 --check
python tools/benchmark/run.py --scenario weekday_peak --iterations 100
python tools/migration-tools/check.py
python -m unittest discover -s tools/tests -v
```

Scenario creation refuses to overwrite an existing scenario unless `--force` is explicit. Vector exports are derived from the fictional dataset and are labeled synthetic. Benchmarks report machine-specific timing without pass/fail latency thresholds. Migration checks are static safety lint; they do not apply migrations. Apply database changes through the backend migration runner after review.
