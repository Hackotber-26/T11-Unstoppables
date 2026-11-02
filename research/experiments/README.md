# Experiments

`pooling_sensitivity.py` runs deterministic policy comparisons over the generated synthetic fixture set. It groups by exact station and destination corridor, constrains party seats and wait windows, and only assigns accessibility-required groups to accessible vehicles.

Run the dataset builder first, then execute from the repository root:

```powershell
python datasets/scripts/build.py build
python research/experiments/pooling_sensitivity.py
python research/experiments/pooling_sensitivity.py --check
python -m unittest discover -s research/tests -v
```

The wait value is a request-window proxy, not a predicted vehicle arrival. The experiment does not model road detours, route sequencing, driver shift rules, safety outcomes, or real demand. Results are exploratory and synthetic-only.