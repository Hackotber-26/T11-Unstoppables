# Research Overview

The research directory captures experiments, analytical work, and exploratory outputs that support platform learning and improvement.

---

## Included areas

- experiments/ — exploratory testing and prototype evaluation
- exports/ — generated outputs for analysis and reporting
- notebooks/ — analysis workflows and research artifacts

---

## Purpose

Research helps the team validate assumptions around demand modeling, pricing, pooling efficiency, and route optimization before production decisions are made.

---

## Expected use

- test new matching or routing ideas
- assess scenario performance under different assumptions
- produce evidence for architecture or product decisions
- support operational and business case analysis

## Reproducible experiments

The current experiment compares deterministic pooling policies using only the explicitly synthetic fixtures under `datasets/synthetic/`. It reports request-level pooling, waiting-time proxy, occupancy, and unserved demand. It does not estimate road detours, predict real demand, or validate safety outcomes.

From the repository root:

```powershell
python datasets/scripts/build.py build
python research/experiments/pooling_sensitivity.py
python -m unittest discover -s research/tests -v
```

Generated outputs are written under `research/exports/`. They are replaceable artifacts and should not be used as evidence for deployment or pricing decisions without real, permitted, independently validated data.
