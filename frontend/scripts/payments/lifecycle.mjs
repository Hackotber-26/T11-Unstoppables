const TRANSITIONS = Object.freeze({
  pending: new Set(["authorized", "failed"]),
  authorized: new Set(["captured", "voided"]),
  captured: new Set(["refunded"]),
  failed: new Set(),
  voided: new Set(),
  refunded: new Set(),
});

export function transitionPayment(payment, nextStatus, eventId) {
  if (!payment || !TRANSITIONS[payment.status]) throw new TypeError("payment has an unknown status");
  if (typeof eventId !== "string" || eventId.length === 0) throw new TypeError("eventId is required for idempotency");
  if (payment.processedEventIds?.includes(eventId)) return payment;
  if (!TRANSITIONS[payment.status].has(nextStatus)) {
    throw new Error(`invalid payment transition: ${payment.status} -> ${nextStatus}`);
  }
  return {
    ...payment,
    status: nextStatus,
    processedEventIds: [...(payment.processedEventIds ?? []), eventId],
  };
}