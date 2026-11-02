import test from "node:test";
import assert from "node:assert/strict";
import { normalizeRouteGeometry } from "../../scripts/core/route-map.mjs";

test("route geometry converts GeoJSON longitude-latitude pairs to map latitude-longitude pairs", () => {
  assert.deepEqual(normalizeRouteGeometry({
    type: "LineString",
    coordinates: [[77.5946, 12.9716], [77.6, 12.98]],
  }), [[12.9716, 77.5946], [12.98, 77.6]]);
});

test("route geometry rejects unsupported, empty, and oversized geometries", () => {
  for (const geometry of [
    { type: "MultiLineString", coordinates: [[[77, 12], [78, 13]]] },
    { type: "LineString", coordinates: [[77, 12]] },
    { type: "LineString", coordinates: Array.from({ length: 20001 }, () => [77, 12]) },
  ]) {
    assert.throws(() => normalizeRouteGeometry(geometry), TypeError);
  }
});

test("route geometry rejects malformed and out-of-range coordinates", () => {
  for (const coordinates of [
    [[77, 12], [Number.NaN, 13]],
    [[181, 12], [77, 13]],
    [[77, -91], [78, 13]],
    [[77], [78, 13]],
  ]) {
    assert.throws(() => normalizeRouteGeometry({ type: "LineString", coordinates }), TypeError);
  }
});
