#!/usr/bin/env python3
"""Convert the synthetic station/edge CSVs into a validated vector graph."""

import argparse
import csv
import heapq
import json
import math
import sys
from pathlib import Path
from typing import Any


ROOT = Path(__file__).resolve().parents[2]
STATIONS = ROOT / "datasets" / "stations" / "synthetic_stations.csv"
ROUTES = ROOT / "datasets" / "routes" / "synthetic_network.csv"
CONTRACT = ROOT / "spec" / "vectors" / "vector-contract.json"


def read_csv(path: Path) -> list[dict[str, str]]:
    with path.open(newline="", encoding="utf-8") as source:
        return list(csv.DictReader(source))


def build_graph() -> dict[str, Any]:
    contract = json.loads(CONTRACT.read_text(encoding="utf-8"))
    station_rows = read_csv(STATIONS)
    route_rows = read_csv(ROUTES)
    nodes = []
    node_ids = set()
    for row in station_rows:
        node_id = row["station_id"]
        x, y = float(row["x_m"]), float(row["y_m"])
        if not node_id or node_id in node_ids or not math.isfinite(x) or not math.isfinite(y) or row["data_class"] != "synthetic":
            raise ValueError(f"invalid or duplicate station vector: {node_id}")
        node_ids.add(node_id)
        nodes.append({"id": node_id, "corridorId": row["corridor_id"], "x": x, "y": y, "accessible": row["accessible"] == "true"})
    edges = []
    edge_ids = set()
    adjacency = {node_id: set() for node_id in node_ids}
    for row in route_rows:
        edge_id = row["edge_id"]
        distance = int(row["distance_meters"])
        seconds = int(row["travel_seconds"])
        if edge_id in edge_ids or row["from_station_id"] not in node_ids or row["to_station_id"] not in node_ids:
            raise ValueError(f"invalid or duplicate route vector: {edge_id}")
        if distance <= 0 or seconds <= 0 or row["data_class"] != "synthetic":
            raise ValueError(f"route edge {edge_id} has invalid cost or classification")
        edge_ids.add(edge_id)
        adjacency[row["from_station_id"]].add(row["to_station_id"])
        edges.append({
            "id": edge_id,
            "from": row["from_station_id"],
            "to": row["to_station_id"],
            "corridorId": row["corridor_id"],
            "distanceMeters": distance,
            "travelSeconds": seconds,
        })
    reachable = set()
    pending = [nodes[0]["id"]] if nodes else []
    while pending:
        node_id = pending.pop()
        if node_id in reachable:
            continue
        reachable.add(node_id)
        pending.extend(adjacency[node_id] - reachable)
    if reachable != node_ids:
        raise ValueError("directed graph does not reach every station")
    return {
        "specVersion": contract["spec_version"],
        "classification": "synthetic",
        "coordinateReference": contract["coordinate_reference"],
        "units": contract["units"],
        "geographicCoordinates": False,
        "nodes": sorted(nodes, key=lambda node: node["id"]),
        "edges": sorted(edges, key=lambda edge: edge["id"]),
    }


def shortest_path(graph: dict[str, Any], start: str, destination: str) -> dict[str, Any]:
    nodes = {node["id"] for node in graph["nodes"]}
    if start not in nodes or destination not in nodes:
        raise ValueError("start and destination must be graph nodes")
    adjacency: dict[str, list[dict[str, Any]]] = {node_id: [] for node_id in nodes}
    for edge in graph["edges"]:
        adjacency[edge["from"]].append(edge)
    best = {start: 0}
    distance_meters = {start: 0}
    previous: dict[str, str] = {}
    queue = [(0, start)]
    while queue:
        seconds, current = heapq.heappop(queue)
        if seconds != best[current]:
            continue
        if current == destination:
            break
        for edge in adjacency[current]:
            candidate = seconds + edge["travelSeconds"]
            if candidate < best.get(edge["to"], float("inf")):
                best[edge["to"]] = candidate
                distance_meters[edge["to"]] = distance_meters[current] + edge["distanceMeters"]
                previous[edge["to"]] = current
                heapq.heappush(queue, (candidate, edge["to"]))
    if destination not in best:
        raise ValueError("no directed route exists")
    path = [destination]
    while path[-1] != start:
        path.append(previous[path[-1]])
    path.reverse()
    return {"nodes": path, "travelSeconds": best[destination], "distanceMeters": distance_meters[destination]}


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", default=str(ROOT / "tools" / "vector-generator" / "generated" / "synthetic-network.json"))
    parser.add_argument("--path", nargs=2, metavar=("START", "DESTINATION"), help="also print shortest route summary")
    parser.add_argument("--check", action="store_true", help="verify an existing export matches current dataset inputs")
    args = parser.parse_args(argv)
    try:
        graph = build_graph()
        if args.path:
            graph["exampleRoute"] = shortest_path(graph, args.path[0], args.path[1])
        output = Path(args.output).resolve()
        serialized = json.dumps(graph, indent=2) + "\n"
        if args.check:
            if not output.is_file() or output.read_text(encoding="utf-8") != serialized:
                raise ValueError(f"vector export is stale or missing: {output}")
        else:
            output.parent.mkdir(parents=True, exist_ok=True)
            output.write_text(serialized, encoding="utf-8")
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError) as error:
        print(f"vector generation failed: {error}", file=sys.stderr)
        return 1
    action = "verified" if args.check else "generated"
    print(f"{action} synthetic graph: nodes={len(graph['nodes'])}, edges={len(graph['edges'])}, output={output}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())