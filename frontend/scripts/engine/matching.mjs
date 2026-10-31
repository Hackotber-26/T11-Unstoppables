const DEFAULT_WEIGHTS = Object.freeze({
  corridor: 0.35,
  time: 0.25,
  detour: 0.2,
  wait: 0.2,
});

function finiteNumber(value, name) {
  if (!Number.isFinite(value)) throw new TypeError(`${name} must be a finite number`);
  return value;
}

function clamp01(value) {
  return Math.max(0, Math.min(1, value));
}

export function scorePoolCandidate(request, candidate, limits = {}) {
  const maxWaitSeconds = limits.maxWaitSeconds ?? 600;
  const maxDetourMeters = limits.maxDetourMeters ?? 2500;
  const maxPickupDeltaSeconds = limits.maxPickupDeltaSeconds ?? 900;
  if (![maxWaitSeconds, maxDetourMeters, maxPickupDeltaSeconds].every((value) => Number.isFinite(value) && value > 0)) {
    throw new RangeError("matching limits must be positive finite numbers");
  }
  const availableSeats = finiteNumber(candidate.availableSeats, "availableSeats");
  const partySize = finiteNumber(request.partySize ?? 1, "partySize");
  if (!Number.isInteger(availableSeats) || !Number.isInteger(partySize) || partySize < 1) {
    throw new RangeError("seat and party counts must be positive integers");
  }
  const waitSeconds = finiteNumber(candidate.waitSeconds, "waitSeconds");
  const detourMeters = finiteNumber(candidate.detourMeters, "detourMeters");
  const pickupDeltaSeconds = finiteNumber(candidate.pickupDeltaSeconds, "pickupDeltaSeconds");
  const corridorSimilarity = clamp01(finiteNumber(candidate.corridorSimilarity, "corridorSimilarity"));
  const reasons = [];

  if (candidate.safetyCompatible === false) reasons.push("safety-incompatible");
  if (availableSeats < partySize) reasons.push("insufficient-capacity");
  if (waitSeconds < 0 || waitSeconds > maxWaitSeconds) reasons.push("wait-limit-exceeded");
  if (detourMeters < 0 || detourMeters > maxDetourMeters) reasons.push("detour-limit-exceeded");
  if (pickupDeltaSeconds < 0 || pickupDeltaSeconds > maxPickupDeltaSeconds) reasons.push("pickup-window-mismatch");

  const waitScore = clamp01(1 - waitSeconds / maxWaitSeconds);
  const detourScore = clamp01(1 - detourMeters / maxDetourMeters);
  const timeScore = clamp01(1 - pickupDeltaSeconds / maxPickupDeltaSeconds);
  const score =
    corridorSimilarity * DEFAULT_WEIGHTS.corridor +
    timeScore * DEFAULT_WEIGHTS.time +
    detourScore * DEFAULT_WEIGHTS.detour +
    waitScore * DEFAULT_WEIGHTS.wait;

  return {
    candidateId: candidate.id,
    eligible: reasons.length === 0,
    score: Math.round(score * 10000) / 10000,
    reasons,
  };
}

export function rankPoolCandidates(request, candidates, limits) {
  return candidates
    .map((candidate) => ({ candidate, ...scorePoolCandidate(request, candidate, limits) }))
    .filter((result) => result.eligible)
    .sort((left, right) => right.score - left.score || String(left.candidateId).localeCompare(String(right.candidateId)));
}