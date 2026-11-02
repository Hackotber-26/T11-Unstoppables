#!/usr/bin/env python3
"""Measure the synthetic pooling policy implementation on a local fixture."""

import argparse
import importlib.util
import json
import math
import statistics
import sys
import time
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
EXPERIMENT_PATH = ROOT / "research" / "experiments" / "pooling_sensitivity.py"
SPEC = importlib.util.spec_from_file_location("pooling_sensitivity", EXPERIMENT_PATH)
EXPERIMENT = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(EXPERIMENT)


def percentile(values: list[int], fraction: float) -> int:
    if not values or not 0 < fraction <= 1:
        raise ValueError("percentile requires values and a fraction in (0, 1]")
    ordered = sorted(values)
    index = max(0, math.ceil(len(ordered) * fraction) - 1)
    return ordered[index]


def run(scenario_id: str, iterations: int, warmup: int) -> dict:
    if iterations < 1 or warmup < 0:
        raise ValueError("iterations must be positive and warmup non-negative")
    scenarios = [
        json.loads(path.read_text(encoding="utf-8"))
        for path in sorted((ROOT / "datasets" / "scenarios").glob("*.json"))
    ]
    scenario = next((item for item in scenarios if item["scenario_id"] == scenario_id), None)
    if scenario is None:
        raise ValueError(f"unknown scenario: {scenario_id}")
    requests = [row for row in EXPERIMENT.load_csv(EXPERIMENT.DATA / "demand_requests.csv") if row["scenario_id"] == scenario_id]
    drivers = [row for row in EXPERIMENT.load_csv(EXPERIMENT.DATA / "driver_supply.csv") if row["scenario_id"] == scenario_id]
    operation = lambda: EXPERIMENT.evaluate(scenario_id, requests, drivers, scenario["max_pool_size"], scenario["max_wait_seconds"])
    for _ in range(warmup):
        operation()
    samples = []
    for _ in range(iterations):
        start = time.perf_counter_ns()
        operation()
        samples.append(time.perf_counter_ns() - start)
    return {
        "classification": "synthetic",
        "scenario_id": scenario_id,
        "request_count": len(requests),
        "iterations": iterations,
        "warmup_iterations": warmup,
        "elapsed_nanoseconds": {
            "median": int(statistics.median(samples)),
            "p95": percentile(samples, 0.95),
            "minimum": min(samples),
            "maximum": max(samples),
        },
        "python_version": sys.version.split()[0],
        "limitations": ["local machine timings are not production capacity guarantees", "only in-memory synthetic pooling is measured"],
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--scenario", default="weekday_peak")
    parser.add_argument("--iterations", type=int, default=100)
    parser.add_argument("--warmup", type=int, default=10)
    parser.add_argument("--output", help="optional JSON result path; defaults to stdout")
    args = parser.parse_args()
    try:
        result = run(args.scenario, args.iterations, args.warmup)
        serialized = json.dumps(result, indent=2) + "\n"
        if args.output:
            output = Path(args.output).resolve()
            output.parent.mkdir(parents=True, exist_ok=True)
            output.write_text(serialized, encoding="utf-8")
            print(f"benchmark result written: {output}")
        else:
            print(serialized, end="")
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError) as error:
        print(f"benchmark failed: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())