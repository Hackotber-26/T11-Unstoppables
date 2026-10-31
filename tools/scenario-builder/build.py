#!/usr/bin/env python3
"""Create a scenario JSON file by safely deriving it from a validated template."""

import argparse
import importlib.util
import json
import re
import sys
from pathlib import Path
from typing import Any


ROOT = Path(__file__).resolve().parents[2]
SCENARIO_DIR = ROOT / "datasets" / "scenarios"
DATASET_VALIDATOR_PATH = ROOT / "datasets" / "scripts" / "build.py"
SPEC = importlib.util.spec_from_file_location("dataset_builder", DATASET_VALIDATOR_PATH)
DATASET_VALIDATOR = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(DATASET_VALIDATOR)


def build_scenario(template_path: Path, scenario_id: str, seed: int, rider_count: int | None = None, description: str | None = None) -> dict[str, Any]:
    if not re.fullmatch(r"[a-z][a-z0-9_]*", scenario_id):
        raise ValueError("scenario id must use lowercase letters, digits, and underscores")
    scenario = json.loads(template_path.read_text(encoding="utf-8"))
    scenario["scenario_id"] = scenario_id
    scenario["description"] = description or f"Derived synthetic scenario from {template_path.stem}."
    scenario["seed"] = seed
    if rider_count is not None:
        scenario["rider_count"] = rider_count
    DATASET_VALIDATOR.validate_scenario(scenario)
    return scenario


def write_scenario(scenario: dict[str, Any], output_path: Path, force: bool = False) -> Path:
    target = output_path.resolve()
    if target.parent != SCENARIO_DIR.resolve() or target.suffix != ".json":
        raise ValueError("output must be a .json file directly under datasets/scenarios")
    if target.exists() and not force:
        raise FileExistsError(f"refusing to overwrite {target}; pass --force to replace it")
    temporary = target.with_suffix(".json.tmp")
    temporary.write_text(json.dumps(scenario, indent=2) + "\n", encoding="utf-8")
    temporary.replace(target)
    return target


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--id", required=True, dest="scenario_id")
    parser.add_argument("--seed", required=True, type=int)
    parser.add_argument("--riders", type=int)
    parser.add_argument("--description")
    parser.add_argument("--template", default="weekday_peak")
    parser.add_argument("--output")
    parser.add_argument("--force", action="store_true")
    args = parser.parse_args(argv)
    try:
        template_path = SCENARIO_DIR / f"{args.template}.json"
        if not template_path.is_file():
            raise FileNotFoundError(f"scenario template not found: {template_path.name}")
        scenario = build_scenario(template_path, args.scenario_id, args.seed, args.riders, args.description)
        output = Path(args.output) if args.output else SCENARIO_DIR / f"{args.scenario_id}.json"
        written = write_scenario(scenario, output, args.force)
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError) as error:
        print(f"scenario build failed: {error}", file=sys.stderr)
        return 1
    print(f"created {written.relative_to(ROOT)} with seed={scenario['seed']} riders={scenario['rider_count']}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())