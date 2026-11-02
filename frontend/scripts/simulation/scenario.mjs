function createRandom(seed) {
  let state = seed >>> 0;
  return () => {
    state = (1664525 * state + 1013904223) >>> 0;
    return state / 0x100000000;
  };
}

export function simulateDemand({
  riders,
  seed = 1,
  corridorCount = 4,
  windowSeconds = 600,
  maxWaitSeconds = 300,
  maxPoolSize = 4,
}) {
  if (!Number.isInteger(riders) || riders < 0) throw new RangeError("riders must be a non-negative integer");
  if (!Number.isInteger(corridorCount) || corridorCount < 1) throw new RangeError("corridorCount must be a positive integer");
  if (!Number.isFinite(windowSeconds) || windowSeconds <= 0) throw new RangeError("windowSeconds must be positive");
  if (!Number.isFinite(maxWaitSeconds) || maxWaitSeconds < 0) throw new RangeError("maxWaitSeconds must be non-negative");
  if (!Number.isInteger(maxPoolSize) || maxPoolSize < 2) throw new RangeError("maxPoolSize must be at least 2");

  const random = createRandom(seed);
  const demand = Array.from({ length: riders }, (_, index) => ({
    id: `rider-${index + 1}`,
    corridor: Math.floor(random() * corridorCount),
    requestedAt: Math.floor(random() * windowSeconds),
  }));
  const corridors = new Map();
  for (const rider of demand) {
    if (!corridors.has(rider.corridor)) corridors.set(rider.corridor, []);
    corridors.get(rider.corridor).push(rider);
  }
  const groups = [];
  for (const corridorDemand of corridors.values()) {
    corridorDemand.sort((left, right) => left.requestedAt - right.requestedAt);
    let group = [];
    for (const rider of corridorDemand) {
      if (
        group.length === maxPoolSize ||
        (group.length > 0 && rider.requestedAt - group[0].requestedAt > maxWaitSeconds)
      ) {
        groups.push(group);
        group = [];
      }
      group.push(rider);
    }
    if (group.length > 0) groups.push(group);
  }
  const formedPools = groups.filter((group) => group.length > 1);
  const pooledRiders = formedPools.reduce((total, group) => total + group.length, 0);
  return {
    seed: seed >>> 0,
    riderCount: riders,
    poolCount: formedPools.length,
    pooledRiderRate: riders === 0 ? 0 : pooledRiders / riders,
    averagePoolSize: formedPools.length === 0 ? 0 : pooledRiders / formedPools.length,
  };
}