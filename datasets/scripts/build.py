#!/usr/bin/env python3
"""Build and validate deterministic synthetic TrainSync datasets."""

from __future__ import annotations

import argparse
import csv
import io
import json
import math
import random
import sys
from pathlib import Path
from typing import Any


ROOT = Path(__file__).resolve().parents[1]
STATIONS_PATH = ROOT / "stations" / "synthetic_stations.csv"
ROUTES_PATH = ROOT / "routes" / "synthetic_network.csv"
SCENARIOS_PATH = ROOT / "scenarios"
OUTPUTS = {
    "demand_requests.csv": [
        "request_id", "scenario_id", "station_id", "destination_corridor_id",
        "requested_offset_seconds", "party_size", "max_wait_seconds",
        "max_detour_meters", "accessibility_required", "data_class",
    ],
    "driver_supply.csv": [
        "driver_id", "scenario_id", "capacity", "accessible_vehicle", "available", "data_class",
    ],
    "safety_events.csv": [
        "event_id", "scenario_id", "request_id", "severity", "event_type", "data_class",
    ],
}
PROBABILITY_FIELDS = (
    "destination_mismatch_rate", "party_size_two_rate",
    "accessible_request_rate", "accessible_vehicle_rate",
)


def read_csv(path: Path, expected_fields: list[str] | None = None) -> list[dict[str, str]]:
    with path.open(newline="", encoding="utf-8") as source:
        reader = csv.DictReader(source)
        if expected_fields is not None and reader.fieldnames != expected_fields:
            raise ValueError(f"{path.relative_to(ROOT)} has unexpected CSV columns")
        return list(reader)


def parse_bool(value: str, field: str) -> bool:
    if value not in {"true", "false"}:
        raise ValueError(f"{field} must be true or false")
    return value == "true"


def load_scenarios() -> list[dict[str, Any]]:
    scenarios = []
    for path in sorted(SCENARIOS_PATH.glob("*.json")):
        scenario = json.loads(path.read_text(encoding="utf-8"))
        if scenario.get("scenario_id") != path.stem:
            raise ValueError(f"scenario id must match filename: {path.name}")
        scenarios.append(scenario)
    if not scenarios:
        raise ValueError("no scenario definitions found")
    return scenarios


def validate_scenario(scenario: dict[str, Any]) -> None:
    required = (
        "scenario_id", "description", "seed", "rider_count", "arrival_window_seconds",
        "arrival_shape", "destination_mismatch_rate", "party_size_two_rate",
        "accessible_request_rate", "safety_event_count", "drivers", "vehicle_capacity",
        "accessible_vehicle_rate", "max_wait_seconds", "max_detour_meters",
        "max_pool_size", "pool_discount_bps",
    )
    missing = [field for field in required if field not in scenario]
    if missing:
        raise ValueError(f"scenario is missing fields: {', '.join(missing)}")
    if not isinstance(scenario["scenario_id"], str) or not scenario["scenario_id"] or not isinstance(scenario["description"], str) or not scenario["description"].strip() or scenario["arrival_shape"] not in {"flat", "peak"}:
        raise ValueError("scenario id or arrival shape is invalid")
    if not isinstance(scenario["seed"], int) or scenario["seed"] < 0:
        raise ValueError("seed must be a non-negative integer")
    for field in PROBABILITY_FIELDS:
        if not 0 <= scenario[field] <= 1:
            raise ValueError(f"{field} must be between 0 and 1")
    for field in ("rider_count", "arrival_window_seconds", "vehicle_capacity", "max_pool_size"):
        if not isinstance(scenario[field], int) or scenario[field] < 1:
            raise ValueError(f"{field} must be a positive integer")
    for field in ("drivers", "safety_event_count", "max_wait_seconds", "max_detour_meters", "pool_discount_bps"):
        if not isinstance(scenario[field], int) or scenario[field] < 0:
            raise ValueError(f"{field} must be a non-negative integer")
    if scenario["max_pool_size"] > scenario["vehicle_capacity"]:
        raise ValueError("max_pool_size cannot exceed vehicle_capacity")
    if scenario["pool_discount_bps"] > 10000:
        raise ValueError("pool_discount_bps cannot exceed 10000")


def generate_outputs() -> dict[str, str]:
    stations = read_csv(STATIONS_PATH)
    station_ids = {row["station_id"] for row in stations}
    corridors = sorted({row["corridor_id"] for row in stations})
    if not stations or len(station_ids) != len(stations) or len(corridors) < 2:
        raise ValueError("station source must contain unique stations in at least two corridors")

    generated: dict[str, list[dict[str, str]]] = {name: [] for name in OUTPUTS}
    for scenario in load_scenarios():
        validate_scenario(scenario)
        rng = random.Random(scenario["seed"])
        scenario_id = scenario["scenario_id"]
        requests: list[dict[str, str]] = []
        for index in range(scenario["rider_count"]):
            station = stations[rng.randrange(len(stations))]
            origin_corridor = station["corridor_id"]
            destination_corridor = origin_corridor
            if rng.random() < scenario["destination_mismatch_rate"]:
                alternatives = [corridor for corridor in corridors if corridor != origin_corridor]
                destination_corridor = alternatives[rng.randrange(len(alternatives))]
            if scenario["arrival_shape"] == "peak":
                arrival = int(rng.triangular(0, scenario["arrival_window_seconds"], scenario["arrival_window_seconds"] * 0.5))
            else:
                arrival = rng.randrange(scenario["arrival_window_seconds"])
            requests.append({
                "request_id": f"{scenario_id}-R{index + 1:05d}",
                "scenario_id": scenario_id,
                "station_id": station["station_id"],
                "destination_corridor_id": destination_corridor,
                "requested_offset_seconds": str(arrival),
                "party_size": "2" if rng.random() < scenario["party_size_two_rate"] else "1",
                "max_wait_seconds": str(scenario["max_wait_seconds"]),
                "max_detour_meters": str(scenario["max_detour_meters"]),
                "accessibility_required": str(rng.random() < scenario["accessible_request_rate"]).lower(),
                "data_class": "synthetic",
            })
        generated["demand_requests.csv"].extend(requests)

        for index in range(scenario["drivers"]):
            generated["driver_supply.csv"].append({
                "driver_id": f"{scenario_id}-D{index + 1:04d}",
                "scenario_id": scenario_id,
                "capacity": str(scenario["vehicle_capacity"]),
                "accessible_vehicle": str(rng.random() < scenario["accessible_vehicle_rate"]).lower(),
                "available": "true",
                "data_class": "synthetic",
            })

        severities = ("low", "medium", "high")
        for index in range(scenario["safety_event_count"]):
            request = requests[rng.randrange(len(requests))]
            generated["safety_events.csv"].append({
                "event_id": f"{scenario_id}-S{index + 1:04d}",
                "scenario_id": scenario_id,
                "request_id": request["request_id"],
                "severity": severities[rng.randrange(len(severities))],
                "event_type": "synthetic_review_required",
                "data_class": "synthetic",
            })

    return {name: serialize_csv(OUTPUTS[name], rows) for name, rows in generated.items()}


def serialize_csv(fields: list[str], rows: list[dict[str, str]]) -> str:
    output = io.StringIO(newline="")
    writer = csv.DictWriter(output, fieldnames=fields, lineterminator="\n", extrasaction="raise")
    writer.writeheader()
    writer.writerows(rows)
    return output.getvalue()


def validate_generated() -> dict[str, int]:
    station_fields = ["station_id", "name", "corridor_id", "x_m", "y_m", "accessible", "active", "data_class"]
    route_fields = ["edge_id", "from_station_id", "to_station_id", "corridor_id", "distance_meters", "travel_seconds", "edge_type", "data_class"]
    stations = read_csv(STATIONS_PATH, station_fields)
    station_ids = {row["station_id"] for row in stations}
    if not stations or len(station_ids) != len(stations):
        raise ValueError("station ids must be non-empty and unique")
    corridors = {row["corridor_id"] for row in stations}
    for station in stations:
        if not station["station_id"] or not station["name"] or not station["corridor_id"] or station["data_class"] != "synthetic":
            raise ValueError("station identity or classification is invalid")
        for field in ("x_m", "y_m"):
            if not math.isfinite(float(station[field])):
                raise ValueError(f"station {station['station_id']} has invalid {field}")
        parse_bool(station["accessible"], "station.accessible")
        parse_bool(station["active"], "station.active")

    route_edges = read_csv(ROUTES_PATH, route_fields)
    edge_ids = {row["edge_id"] for row in route_edges}
    graph = {station_id: set() for station_id in station_ids}
    for edge in route_edges:
        if not edge["edge_id"] or edge["from_station_id"] not in station_ids or edge["to_station_id"] not in station_ids:
            raise ValueError(f"route {edge['edge_id']} references an unknown station")
        if edge["data_class"] != "synthetic" or edge["from_station_id"] == edge["to_station_id"]:
            raise ValueError(f"route {edge['edge_id']} identity or classification is invalid")
        if edge["corridor_id"] not in corridors | {"COR-X"}:
            raise ValueError(f"route {edge['edge_id']} references an unknown corridor")
        if int(edge["distance_meters"]) <= 0 or int(edge["travel_seconds"]) <= 0:
            raise ValueError(f"route {edge['edge_id']} has non-positive distance or travel time")
        graph[edge["from_station_id"]].add(edge["to_station_id"])
    if len(edge_ids) != len(route_edges):
        raise ValueError("route edge ids must be unique")
    reached = set()
    pending = [next(iter(station_ids))] if station_ids else []
    while pending:
        station_id = pending.pop()
        if station_id in reached:
            continue
        reached.add(station_id)
        pending.extend(graph[station_id] - reached)
    if reached != station_ids:
        raise ValueError("directed route graph must reach every station from the first station")

    scenarios = load_scenarios()
    scenario_ids = {scenario["scenario_id"] for scenario in scenarios}
    if len(scenario_ids) != len(scenarios):
        raise ValueError("scenario ids must be unique")
    for scenario in scenarios:
        validate_scenario(scenario)

    demand = read_csv(ROOT / "synthetic" / "demand_requests.csv", OUTPUTS["demand_requests.csv"])
    drivers = read_csv(ROOT / "synthetic" / "driver_supply.csv", OUTPUTS["driver_supply.csv"])
    events = read_csv(ROOT / "synthetic" / "safety_events.csv", OUTPUTS["safety_events.csv"])
    request_ids = {row["request_id"] for row in demand}
    if len(request_ids) != len(demand):
        raise ValueError("request ids must be unique")
    if any(row["scenario_id"] not in scenario_ids or row["data_class"] != "synthetic" for row in demand + drivers + events):
        raise ValueError("generated record has an unknown scenario or incorrect data classification")
    scenario_by_id = {scenario["scenario_id"]: scenario for scenario in scenarios}
    demand_counts = {scenario_id: 0 for scenario_id in scenario_ids}
    for row in demand:
        scenario = scenario_by_id[row["scenario_id"]]
        if row["station_id"] not in station_ids or row["destination_corridor_id"] not in corridors:
            raise ValueError(f"demand request {row['request_id']} references an unknown station or corridor")
        if not 0 <= int(row["requested_offset_seconds"]) < scenario["arrival_window_seconds"]:
            raise ValueError(f"demand request {row['request_id']} is outside its arrival window")
        if int(row["party_size"]) not in (1, 2) or int(row["party_size"]) > scenario["max_pool_size"]:
            raise ValueError(f"demand request {row['request_id']} has invalid party size")
        if int(row["max_wait_seconds"]) != scenario["max_wait_seconds"] or int(row["max_detour_meters"]) != scenario["max_detour_meters"]:
            raise ValueError(f"demand request {row['request_id']} does not match scenario constraints")
        parse_bool(row["accessibility_required"], "demand.accessibility_required")
        demand_counts[row["scenario_id"]] += 1

    driver_ids = {row["driver_id"] for row in drivers}
    if len(driver_ids) != len(drivers):
        raise ValueError("driver ids must be unique")
    driver_counts = {scenario_id: 0 for scenario_id in scenario_ids}
    for row in drivers:
        scenario = scenario_by_id[row["scenario_id"]]
        if int(row["capacity"]) != scenario["vehicle_capacity"]:
            raise ValueError(f"driver {row['driver_id']} capacity does not match its scenario")
        parse_bool(row["accessible_vehicle"], "driver.accessible_vehicle")
        parse_bool(row["available"], "driver.available")
        driver_counts[row["scenario_id"]] += 1

    event_ids = {row["event_id"] for row in events}
    if len(event_ids) != len(events):
        raise ValueError("safety event ids must be unique")
    event_counts = {scenario_id: 0 for scenario_id in scenario_ids}
    request_scenarios = {row["request_id"]: row["scenario_id"] for row in demand}
    for row in events:
        if row["request_id"] not in request_ids or request_scenarios[row["request_id"]] != row["scenario_id"]:
            raise ValueError(f"safety event {row['event_id']} references an unknown or cross-scenario request")
        if row["severity"] not in {"low", "medium", "high"} or row["event_type"] != "synthetic_review_required":
            raise ValueError(f"safety event {row['event_id']} contains invalid event fields")
        event_counts[row["scenario_id"]] += 1
    for row in demand:
        if int(row["requested_offset_seconds"]) < 0 or int(row["party_size"]) not in (1, 2):
            raise ValueError(f"demand request {row['request_id']} has invalid timing or party size")
    expected_demand = sum(scenario["rider_count"] for scenario in scenarios)
    expected_drivers = sum(scenario["drivers"] for scenario in scenarios)
    expected_events = sum(scenario["safety_event_count"] for scenario in scenarios)
    if (len(demand), len(drivers), len(events)) != (expected_demand, expected_drivers, expected_events):
        raise ValueError("generated row counts do not match scenario definitions")
    if any(demand_counts[s["scenario_id"]] != s["rider_count"] or driver_counts[s["scenario_id"]] != s["drivers"] or event_counts[s["scenario_id"]] != s["safety_event_count"] for s in scenarios):
        raise ValueError("generated per-scenario counts do not match scenario definitions")
    return {"stations": len(stations), "route_edges": len(route_edges), "scenarios": len(scenarios), "demand_requests": len(demand), "drivers": len(drivers), "safety_events": len(events)}


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", choices=("build", "validate", "check"))
    args = parser.parse_args(argv)
    try:
        if args.command == "build":
            for name, contents in generate_outputs().items():
                (ROOT / "synthetic" / name).write_text(contents, encoding="utf-8", newline="")
        elif args.command == "check":
            for name, contents in generate_outputs().items():
                path = ROOT / "synthetic" / name
                if not path.exists() or path.read_text(encoding="utf-8") != contents:
                    raise ValueError(f"generated file is stale: {path.relative_to(ROOT)}")
        counts = validate_generated()
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError) as error:
        print(f"dataset validation failed: {error}", file=sys.stderr)
        return 1
    print("dataset validation passed: " + ", ".join(f"{name}={count}" for name, count in counts.items()))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())