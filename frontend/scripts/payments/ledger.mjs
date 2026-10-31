export function calculateWalletBalance(entries) {
  const seen = new Set();
  let balanceCents = 0;
  for (const entry of entries) {
    if (typeof entry.id !== "string" || entry.id.length === 0) throw new TypeError("ledger entry id is required");
    if (seen.has(entry.id)) continue;
    if (!Number.isSafeInteger(entry.amountCents) || entry.amountCents === 0) {
      throw new TypeError("amountCents must be a non-zero safe integer");
    }
    seen.add(entry.id);
    balanceCents += entry.amountCents;
    if (!Number.isSafeInteger(balanceCents)) throw new RangeError("wallet balance exceeds safe integer range");
    if (balanceCents < 0) throw new RangeError("ledger entries would overdraw the wallet");
  }
  return balanceCents;
}