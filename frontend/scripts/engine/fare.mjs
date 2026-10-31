export function calculateFare({
  baseCents,
  distanceMeters,
  ratePerKmCents,
  pooledDiscountBps = 0,
  minimumFareCents = 0,
}) {
  for (const [name, value] of Object.entries({ baseCents, distanceMeters, ratePerKmCents, pooledDiscountBps, minimumFareCents })) {
    if (!Number.isFinite(value) || value < 0) throw new RangeError(`${name} must be a non-negative finite number`);
  }
  for (const [name, value] of Object.entries({ baseCents, ratePerKmCents, pooledDiscountBps, minimumFareCents })) {
    if (!Number.isSafeInteger(value)) throw new TypeError(`${name} must be a safe integer`);
  }
  if (pooledDiscountBps > 10000) throw new RangeError("pooledDiscountBps cannot exceed 10000");

  const distanceFareCents = Math.ceil((distanceMeters / 1000) * ratePerKmCents);
  const undiscountedCents = baseCents + distanceFareCents;
  const discountCents = Math.floor((undiscountedCents * pooledDiscountBps) / 10000);
  const totalCents = Math.max(minimumFareCents, undiscountedCents - discountCents);
  if (![distanceFareCents, undiscountedCents, discountCents, totalCents].every(Number.isSafeInteger)) {
    throw new RangeError("fare exceeds safe integer range");
  }
  return { currency: "USD", baseCents, distanceFareCents, discountCents, totalCents };
}