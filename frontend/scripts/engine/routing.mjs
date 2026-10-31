function requireId(value, name) {
  if (typeof value !== "string" || value.length === 0) throw new TypeError(`${name} must be a non-empty string`);
  return value;
}

export function shortestRoute(edges, start, destination) {
  requireId(start, "start");
  requireId(destination, "destination");
  const adjacency = new Map();
  const distances = new Map([[start, 0]]);
  const meters = new Map([[start, 0]]);
  const previous = new Map();
  const unvisited = new Set([start]);

  for (const edge of edges) {
    requireId(edge.from, "edge.from");
    requireId(edge.to, "edge.to");
    if (!Number.isFinite(edge.travelSeconds) || edge.travelSeconds < 0) {
      throw new RangeError("edge.travelSeconds must be a non-negative finite number");
    }
    if (!Number.isFinite(edge.distanceMeters) || edge.distanceMeters < 0) {
      throw new RangeError("edge.distanceMeters must be a non-negative finite number");
    }
    if (!adjacency.has(edge.from)) adjacency.set(edge.from, []);
    adjacency.get(edge.from).push(edge);
  }

  while (unvisited.size > 0) {
    let current;
    let currentDistance = Infinity;
    for (const node of unvisited) {
      const distance = distances.get(node) ?? Infinity;
      if (distance < currentDistance) {
        current = node;
        currentDistance = distance;
      }
    }
    if (current === undefined || currentDistance === Infinity) break;
    unvisited.delete(current);
    if (current === destination) break;

    for (const edge of adjacency.get(current) ?? []) {
      const nextDistance = currentDistance + edge.travelSeconds;
      if (nextDistance < (distances.get(edge.to) ?? Infinity)) {
        distances.set(edge.to, nextDistance);
        meters.set(edge.to, (meters.get(current) ?? 0) + edge.distanceMeters);
        previous.set(edge.to, current);
        unvisited.add(edge.to);
      }
    }
  }

  if (!distances.has(destination)) return null;
  const path = [destination];
  while (path[0] !== start) path.unshift(previous.get(path[0]));
  return {
    nodes: path,
    travelSeconds: distances.get(destination),
    distanceMeters: meters.get(destination),
  };
}

export function estimateEta(route, { dwellSeconds = 0, varianceRatio = 0.15 } = {}) {
  if (!route || !Number.isFinite(route.travelSeconds) || route.travelSeconds < 0) {
    throw new TypeError("route.travelSeconds must be a non-negative finite number");
  }
  if (!Number.isFinite(dwellSeconds) || dwellSeconds < 0) throw new RangeError("dwellSeconds must be non-negative");
  if (!Number.isFinite(varianceRatio) || varianceRatio < 0) throw new RangeError("varianceRatio must be non-negative");
  const expectedSeconds = Math.ceil(route.travelSeconds + dwellSeconds);
  return {
    expectedSeconds,
    lowerSeconds: Math.floor(expectedSeconds * (1 - varianceRatio)),
    upperSeconds: Math.ceil(expectedSeconds * (1 + varianceRatio)),
  };
}