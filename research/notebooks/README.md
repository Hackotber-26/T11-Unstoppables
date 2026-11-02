# Research Notebooks

No notebook is required for the current experiment; the checked-in Python experiment is the source of truth because it is deterministic, testable, and runnable without notebook dependencies.

When adding notebooks, keep them exploratory and synthetic-only by default. Record the dataset classification, input paths, scenario seeds, assumptions, and limitations in the first markdown cell. Move reusable logic into a tested experiment module, clear outputs that contain sensitive information, and export decision-relevant results to `research/exports/` through a reproducible script.