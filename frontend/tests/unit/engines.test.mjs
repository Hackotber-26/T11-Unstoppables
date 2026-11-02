import test from "node:test";
import assert from "node:assert/strict";
import {
  calculateFare,
  calculateWalletBalance,
  estimateEta,
  rankPoolCandidates,
  shortestRoute,
  simulateDemand,
  transitionPayment,
} from "../../scripts/engine/index.mjs";

test("fare uses integer paise and applies pooled discount", () => {
  assert.deepEqual(calculateFare({ basePaise: 3000, distanceMeters: 1500, ratePerKmPaise: 1200, pooledDiscountBps: 1000 }), {
    currency: "INR", basePaise: 3000, distancePaise: 1800, discountPaise: 480, totalPaise: 4320,
  });
  assert.throws(() => calculateFare({ basePaise: 10.5, distanceMeters: 100, ratePerKmPaise: 1 }), /safe integer/);
});

test("pool ranking excludes unsafe and over-limit candidates", () => {
  const candidates = [
    { id: "a", availableSeats: 2, waitSeconds: 100, detourMeters: 200, pickupDeltaSeconds: 90, corridorSimilarity: 0.9 },
    { id: "b", availableSeats: 2, waitSeconds: 700, detourMeters: 200, pickupDeltaSeconds: 90, corridorSimilarity: 1 },
  ];
  assert.deepEqual(rankPoolCandidates({ partySize: 1 }, candidates).map(({ candidateId }) => candidateId), ["a"]);
  assert.throws(() => rankPoolCandidates({ partySize: 1 }, candidates, { maxWaitSeconds: 0 }), /positive finite/);
});

test("routing selects the least-time path and estimates a bounded ETA", () => {
  const route = shortestRoute([
    { from: "A", to: "B", travelSeconds: 20, distanceMeters: 100 },
    { from: "A", to: "C", travelSeconds: 5, distanceMeters: 200 },
    { from: "C", to: "B", travelSeconds: 5, distanceMeters: 200 },
  ], "A", "B");
  assert.deepEqual(route, { nodes: ["A", "C", "B"], travelSeconds: 10, distanceMeters: 400 });
  assert.deepEqual(estimateEta(route, { dwellSeconds: 5 }), { expectedSeconds: 15, lowerSeconds: 12, upperSeconds: 18 });
});

test("simulation is reproducible for a fixed seed", () => {
  assert.deepEqual(simulateDemand({ riders: 100, seed: 42 }), simulateDemand({ riders: 100, seed: 42 }));
});

test("payment lifecycle accepts valid transitions and ignores duplicate events", () => {
  const pending = { id: "payment-1", status: "pending" };
  const authorized = transitionPayment(pending, "authorized", "event-1");
  assert.equal(transitionPayment(authorized, "captured", "event-1"), authorized);
  assert.throws(() => transitionPayment(authorized, "refunded", "event-2"), /invalid payment transition/);
});

test("wallet ledger is idempotent and rejects overdrafts", () => {
  assert.equal(calculateWalletBalance([
    { id: "deposit-1", amountPaise: 1000 },
    { id: "deposit-1", amountPaise: 1000 },
    { id: "fare-1", amountPaise: -250 },
  ]), 750);
  assert.throws(() => calculateWalletBalance([{ id: "fare-2", amountPaise: -1 }]), /overdraw/);
});