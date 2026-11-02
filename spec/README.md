# Spec Overview

The spec directory stores the structured definitions that guide system behavior, pricing assumptions, scenario rules, and operational tuning.

---

## Included areas

- configs/ — operational and service configuration defaults
- scenarios/ — scenario definitions and scheduling assumptions
- tariffs/ — fare and pricing policy rules
- vectors/ — vector or movement definitions used for modeling and simulation

---

## Why this matters

TrainSync is highly sensitive to pricing, timing, and route assumptions. The specification layer makes these assumptions explicit, measurable, and manageable.

---

## Usage

This layer is used by:

- simulation runs
- fare evaluation
- routing and matching tuning
- scenario reproducibility and experimentation
- platform governance and cost modeling

## Versioned contracts

- `configs/runtime-defaults.json` defines local service and algorithm defaults without credentials.
- `tariffs/usd-v1.json` is the versioned USD fare schedule and is checked against the Go fare engine constants.
- `scenarios/scenario.schema.json` defines the scenario input contract used by datasets and experiments.
- `vectors/vector-contract.json` defines the local Cartesian coordinate and directed-edge semantics used by the fictional dataset.
- `validate.py` checks the machine-readable contracts and their cross-file compatibility using Python's standard library.

Run `python spec/validate.py` from the repository root. Specs are versioned contracts; changing one requires review of consumers and regenerated research outputs. No secrets or real rider data belong here.
