export function calculateFare({
  basePaise,
  distanceMeters,
  ratePerKmPaise,
  pooledDiscountBps = 0,
  minimumFarePaise = 0,
}) {
  for (const [name, value] of Object.entries({ basePaise, distanceMeters, ratePerKmPaise, pooledDiscountBps, minimumFarePaise })) {
    if (!Number.isFinite(value) || value < 0) throw new RangeError(`${name} must be a non-negative finite number`);
  }
  for (const [name, value] of Object.entries({ basePaise, ratePerKmPaise, pooledDiscountBps, minimumFarePaise })) {
    if (!Number.isSafeInteger(value)) throw new TypeError(`${name} must be a safe integer`);
  }
  if (pooledDiscountBps > 10000) throw new RangeError("pooledDiscountBps cannot exceed 10000");

  const distancePaise = Math.ceil((distanceMeters / 1000) * ratePerKmPaise);
  const undiscountedPaise = basePaise + distancePaise;
  const discountPaise = Math.floor((undiscountedPaise * pooledDiscountBps) / 10000);
  const totalPaise = Math.max(minimumFarePaise, undiscountedPaise - discountPaise);
  if (![distancePaise, undiscountedPaise, discountPaise, totalPaise].every(Number.isSafeInteger)) {
    throw new RangeError("fare exceeds safe integer range");
  }
  return { currency: "INR", basePaise, distancePaise, discountPaise, totalPaise };
}