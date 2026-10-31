import test from "node:test";
import assert from "node:assert/strict";
import { ApiClient, ApiError } from "../../scripts/core/api.mjs";

test("client sends bearer auth and serializes trip creation", async () => {
  let captured;
  const client = new ApiClient("http://localhost:8080/api/v1", async (url, options) => {
    captured = { url, options };
    return new Response(JSON.stringify({ success: true, data: { id: "trip-1" } }), { status: 201 });
  });
  client.setToken("access-token");
  const trip = await client.createTrip({ originId: "ST-A1", destination: "COR-A", poolMode: "corridor", distanceMeters: 1200 });
  assert.equal(trip.id, "trip-1");
  assert.equal(captured.url, "http://localhost:8080/api/v1/trips");
  assert.equal(captured.options.headers.Authorization, "Bearer access-token");
  assert.deepEqual(JSON.parse(captured.options.body), { originId: "ST-A1", destination: "COR-A", poolMode: "corridor", distanceMeters: 1200 });
});

test("login request never includes a previous bearer token", async () => {
  let captured;
  const client = new ApiClient("http://api.test/api/v1", async (_url, options) => {
    captured = options;
    return new Response(JSON.stringify({ success: true, data: { accessToken: "new-token" } }), { status: 200 });
  });
  client.setToken("old-token");
  const session = await client.login({ email: "rider@example.test", password: "password" });
  assert.equal(session.accessToken, "new-token");
  assert.equal(captured.headers.Authorization, undefined);
});

test("client propagates backend status and message", async () => {
  const client = new ApiClient("http://api.test/api/v1", async () => new Response(
    JSON.stringify({ success: false, message: "invalid email or password" }), { status: 401 },
  ));
  await assert.rejects(client.login({ email: "rider@example.test", password: "bad" }), (error) => {
    assert.ok(error instanceof ApiError);
    assert.equal(error.status, 401);
    assert.equal(error.message, "invalid email or password");
    return true;
  });
});

test("wallet payment includes stable idempotency key", async () => {
  let captured;
  const client = new ApiClient("http://api.test/api/v1", async (_url, options) => {
    captured = options;
    return new Response(JSON.stringify({ success: true, data: { status: "paid-from-wallet" } }), { status: 200 });
  });
  client.setToken("token");
  await client.payTrip("trip/1", "trip-payment-trip/1");
  assert.equal(captured.headers["Idempotency-Key"], "trip-payment-trip/1");
  assert.equal(captured.method, "POST");
});

test("trip list and detail use authenticated backend routes", async () => {
  const requests = [];
  const client = new ApiClient("http://api.test/api/v1", async (url, options) => {
    requests.push({ url, options });
    return new Response(JSON.stringify({ success: true, data: [] }), { status: 200 });
  });
  client.setToken("token");
  await client.listTrips();
  await client.getTrip("trip-id");
  assert.deepEqual(requests.map((request) => request.url), [
    "http://api.test/api/v1/trips",
    "http://api.test/api/v1/trips/trip-id",
  ]);
  assert.ok(requests.every((request) => request.options.headers.Authorization === "Bearer token"));
});