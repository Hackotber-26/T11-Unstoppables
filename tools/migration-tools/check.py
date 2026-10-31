#!/usr/bin/env python3
"""Lint versioned SQL migrations for ordering and destructive statements."""

import argparse
import re
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
NAME_PATTERN = re.compile(r"^(\d{6})_[a-z0-9_]+\.sql$")
DESTRUCTIVE = re.compile(r"(?im)^\s*(DROP\s+(TABLE|DATABASE|COLUMN)|TRUNCATE\b|ALTER\s+TABLE\b[^;]*\bDROP\s+COLUMN)\b")


def check(directory: Path) -> list[str]:
    errors = []
    files = sorted(directory.glob("*.sql"))
    if not files:
        return ["no SQL migrations found"]
    versions = []
    for path in files:
        match = NAME_PATTERN.fullmatch(path.name)
        if not match:
            errors.append(f"invalid migration filename: {path.name}")
            continue
        versions.append((int(match.group(1)), path))
        sql = path.read_text(encoding="utf-8")
        if not sql.strip() or not sql.rstrip().endswith(";"):
            errors.append(f"migration must contain SQL ending in a semicolon: {path.name}")
        if DESTRUCTIVE.search(sql) and "-- destructive-reviewed:" not in sql:
            errors.append(f"destructive SQL requires a -- destructive-reviewed: reference: {path.name}")
    versions.sort()
    numbers = [number for number, _ in versions]
    if len(numbers) != len(set(numbers)):
        errors.append("migration numeric prefixes must be unique")
    if numbers and numbers != list(range(numbers[0], numbers[0] + len(numbers))):
        errors.append("migration numeric prefixes must be contiguous")
    return errors


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--directory", default=str(ROOT / "backend" / "migrations"))
    args = parser.parse_args()
    errors = check(Path(args.directory))
    if errors:
        print("\n".join(errors), file=sys.stderr)
        return 1
    print(f"migration lint passed: {len(list(Path(args.directory).glob('*.sql')))} files")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())