#!/usr/bin/env python3
"""Validate TrainSync versioned specs and their cross-file contracts."""

import json
import re
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parent
REPO = ROOT.parent


def load_json(path: Path):
    return json.loads(path.read_text(encoding="utf-8"))


def validate() -> list[str]:
    errors = []
    defaults = load_json(ROOT / "configs" / "runtime-defaults.json")
    tariff = load_json(ROOT / "tariffs" / "usd-v1.json")
    scenario_schema = load_json(ROOT / "scenarios" / "scenario.schema.json")
    vector_contract = load_json(ROOT / "vectors" / "vector-contract.json")

    weights = defaults["pooling"]["matching_weights"]
    if not abs(sum(weights.values()) - 1.0) < 1e-9:
        errors.append("pooling matching weights must sum to 1")
    if defaults["session"]["lifetime_seconds"] != 3600 or defaults["session"]["minimum_secret_characters"] < 32:
        errors.append("runtime session defaults are inconsistent with the backend auth contract")
    if defaults["data_policy"]["external_paid_apis_allowed"] is not False:
        errors.append("paid external APIs must remain disabled")
    if tariff["currency"] != "USD" or tariff["status"] != "simulation_default":
        errors.append("tariff must remain an explicitly non-production USD simulation default")
    if tariff["base_fare_cents"] != 250 or tariff["distance_rate_cents_per_kilometer"] != 150 or tariff["corridor_discount_basis_points"] != 1500:
        errors.append("USD tariff values differ from backend/internal/fares/engine.go")
    if tariff["maximum_trip_distance_meters"] != 500000:
        errors.append("tariff maximum distance differs from backend fare engine validation")
    if scenario_schema.get("$schema") != "https://json-schema.org/draft/2020-12/schema":
        errors.append("scenario schema draft is not supported contract version")
    if vector_contract["coordinate_reference"] != "synthetic-local-cartesian" or vector_contract["geographic_coordinates"] is not False:
        errors.append("vector contract must not claim real geographic coordinates")

    scenario_ids = set()
    required = set(scenario_schema["required"])
    for path in sorted((REPO / "datasets" / "scenarios").glob("*.json")):
        scenario = load_json(path)
        if set(scenario) != required:
            errors.append(f"{path.name} fields do not match scenario schema")
        if scenario.get("scenario_id") != path.stem or not re.fullmatch(r"[a-z][a-z0-9_]*", path.stem):
            errors.append(f"{path.name} scenario_id must match its filename")
        if path.stem in scenario_ids:
            errors.append(f"duplicate scenario id: {path.stem}")
        scenario_ids.add(path.stem)
        if scenario.get("max_pool_size", 0) > scenario.get("vehicle_capacity", 0):
            errors.append(f"{path.name} pool size exceeds vehicle capacity")
    if not scenario_ids:
        errors.append("at least one scenario definition is required")
    return errors


if __name__ == "__main__":
    try:
        problems = validate()
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError) as error:
        print(f"spec validation failed: {error}", file=sys.stderr)
        raise SystemExit(1)
    if problems:
        print("\n".join(problems), file=sys.stderr)
        raise SystemExit(1)
    print("spec validation passed: runtime, tariff, scenarios, and vector contracts")