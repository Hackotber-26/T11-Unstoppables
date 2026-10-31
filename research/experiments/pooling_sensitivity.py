#!/usr/bin/env python3
"""Compare pooling wait/capacity policies on synthetic fixtures only."""

from __future__ import annotations

import argparse
import csv
import io
import json
from pathlib import Path
from typing import Any


ROOT = Path(__file__).resolve().parents[2]
DATA = ROOT / "datasets" / "synthetic"
SCENARIOS = ROOT / "datasets" / "scenarios"
EXPORTS = ROOT / "research" / "exports"
FIELDS = [
    "scenario_id", "max_pool_size", "max_wait_seconds", "requests_total",
    "requests_served", "requests_unserved", "pooled_requests",
    "pool_formation_rate", "pool_count", "average_occupancy_seats",
    "average_wait_seconds_proxy", "maximum_wait_seconds_proxy",
    "accessible_requests_unserved",
]


def load_csv(path: Path) -> list[dict[str, str]]:
    with path.open(newline="", encoding="utf-8") as source:
        return list(csv.DictReader(source))


def form_groups(requests: list[dict[str, str]], capacity: int, max_wait: int) -> list[list[dict[str, str]]]:
    ordered = sorted(requests, key=lambda row: (
        row["station_id"], row["destination_corridor_id"], int(row["requested_offset_seconds"]), row["request_id"],
    ))
    groups: list[list[dict[str, str]]] = []
    current: list[dict[str, str]] = []
    seats = 0

    def flush() -> None:
        nonlocal current, seats
        if current:
            groups.append(current)
        current = []
        seats = 0

    for request in ordered:
        request_seats = int(request["party_size"])
        same_corridor = bool(current) and (
            current[0]["station_id"] == request["station_id"]
            and current[0]["destination_corridor_id"] == request["destination_corridor_id"]
        )
        inside_window = same_corridor and (
            int(request["requested_offset_seconds"]) - int(current[0]["requested_offset_seconds"]) <= max_wait
        )
        if current and (not inside_window or seats + request_seats > capacity):
            flush()
        current.append(request)
        seats += request_seats
    flush()
    return groups


def evaluate(
    scenario_id: str,
    requests: list[dict[str, str]],
    drivers: list[dict[str, str]],
    capacity: int,
    max_wait: int,
) -> dict[str, Any]:
    groups = form_groups(requests, capacity, max_wait)
    available = [driver for driver in drivers if driver["available"] == "true"]
    accessible_available = sum(driver["accessible_vehicle"] == "true" for driver in available)
    non_accessible_available = len(available) - accessible_available
    served_requests = 0
    pooled_requests = 0
    pool_count = 0
    served_seats = 0
    wait_total = 0
    wait_count = 0
    max_observed_wait = 0
    accessible_unserved = 0

    for group in sorted(groups, key=lambda rows: (int(rows[0]["requested_offset_seconds"]), rows[0]["request_id"])):
        needs_accessible = any(row["accessibility_required"] == "true" for row in group)
        if needs_accessible:
            if accessible_available == 0:
                accessible_unserved += sum(row["accessibility_required"] == "true" for row in group)
                continue
            accessible_available -= 1
        elif non_accessible_available > 0:
            non_accessible_available -= 1
        elif accessible_available > 0:
            accessible_available -= 1
        else:
            continue

        served_requests += len(group)
        served_seats += sum(int(row["party_size"]) for row in group)
        if len(group) > 1:
            pool_count += 1
            pooled_requests += len(group)
        latest_pickup = max(int(row["requested_offset_seconds"]) for row in group)
        for row in group:
            wait = latest_pickup - int(row["requested_offset_seconds"])
            wait_total += wait
            wait_count += 1
            max_observed_wait = max(max_observed_wait, wait)

    total = len(requests)
    return {
        "scenario_id": scenario_id,
        "max_pool_size": capacity,
        "max_wait_seconds": max_wait,
        "requests_total": total,
        "requests_served": served_requests,
        "requests_unserved": total - served_requests,
        "pooled_requests": pooled_requests,
        "pool_formation_rate": pooled_requests / served_requests if served_requests else 0.0,
        "pool_count": pool_count,
        "average_occupancy_seats": served_seats / (served_requests - pooled_requests + pool_count) if served_requests else 0.0,
        "average_wait_seconds_proxy": wait_total / wait_count if wait_count else 0.0,
        "maximum_wait_seconds_proxy": max_observed_wait,
        "accessible_requests_unserved": accessible_unserved,
    }


def analyze_all() -> list[dict[str, Any]]:
    demand = load_csv(DATA / "demand_requests.csv")
    driver_supply = load_csv(DATA / "driver_supply.csv")
    rows = []
    for path in sorted(SCENARIOS.glob("*.json")):
        scenario = json.loads(path.read_text(encoding="utf-8"))
        scenario_id = scenario["scenario_id"]
        if scenario_id != path.stem:
            raise ValueError(f"scenario id does not match {path.name}")
        scenario_requests = [row for row in demand if row["scenario_id"] == scenario_id]
        scenario_drivers = [row for row in driver_supply if row["scenario_id"] == scenario_id]
        capacities = sorted({min(2, scenario["vehicle_capacity"]), min(3, scenario["vehicle_capacity"]), min(scenario["max_pool_size"], scenario["vehicle_capacity"])})
        waits = sorted({min(120, scenario["max_wait_seconds"]), min(300, scenario["max_wait_seconds"]), scenario["max_wait_seconds"]})
        for capacity in capacities:
            for wait in waits:
                rows.append(evaluate(scenario_id, scenario_requests, scenario_drivers, capacity, wait))
    return rows


def render_exports(rows: list[dict[str, Any]]) -> dict[str, str]:
    table = io.StringIO(newline="")
    writer = csv.DictWriter(table, fieldnames=FIELDS, lineterminator="\n")
    writer.writeheader()
    writer.writerows(rows)
    report = {
        "classification": "synthetic",
        "method": "greedy exact-origin/destination grouping with party-seat capacity and configured pickup wait",
        "limitations": [
            "wait is modeled as earliest-to-latest group request time, not vehicle ETA",
            "detour and network travel time are not evaluated",
            "synthetic driver availability does not represent a real fleet",
            "pooling rate is pooled requests divided by served requests",
        ],
        "runs": len(rows),
        "scenarios": sorted({row["scenario_id"] for row in rows}),
    }
    return {
        "pooling_sensitivity.csv": table.getvalue(),
        "pooling_sensitivity.json": json.dumps(report, indent=2) + "\n",
    }


def export(rows: list[dict[str, Any]]) -> None:
    EXPORTS.mkdir(parents=True, exist_ok=True)
    for name, contents in render_exports(rows).items():
        (EXPORTS / name).write_text(contents, encoding="utf-8", newline="")


def check_exports(rows: list[dict[str, Any]]) -> None:
    for name, contents in render_exports(rows).items():
        path = EXPORTS / name
        if not path.exists() or path.read_text(encoding="utf-8") != contents:
            raise ValueError(f"research export is stale: {path.relative_to(ROOT)}")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true", help="verify checked-in exports match current inputs")
    args = parser.parse_args()
    rows = analyze_all()
    if not rows:
        raise SystemExit("no scenario runs were produced")
    if args.check:
        check_exports(rows)
    else:
        export(rows)
    print(f"research experiment passed: scenarios={len({row['scenario_id'] for row in rows})}, runs={len(rows)}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())