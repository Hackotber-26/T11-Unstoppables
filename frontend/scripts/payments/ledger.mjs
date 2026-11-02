export function calculateWalletBalance(entries) {
  const seen = new Set();
  let balancePaise = 0;
  for (const entry of entries) {
    if (typeof entry.id !== "string" || entry.id.length === 0) throw new TypeError("ledger entry id is required");
    if (seen.has(entry.id)) continue;
    if (!Number.isSafeInteger(entry.amountPaise) || entry.amountPaise === 0) {
      throw new TypeError("amountPaise must be a non-zero safe integer");
    }
    seen.add(entry.id);
    balancePaise += entry.amountPaise;
    if (!Number.isSafeInteger(balancePaise)) throw new RangeError("wallet balance exceeds safe integer range");
    if (balancePaise < 0) throw new RangeError("ledger entries would overdraw the wallet");
  }
  return balancePaise;
}