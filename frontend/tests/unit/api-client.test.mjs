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
  const trip = await client.createTrip({
    originId: "ST-A1", destination: "COR-A", poolMode: "corridor",
    pickupLatitude: 12.97, pickupLongitude: 77.59, dropoffLatitude: 12.98, dropoffLongitude: 77.6,
  });
  assert.equal(trip.id, "trip-1");
  assert.equal(captured.url, "http://localhost:8080/api/v1/trips");
  assert.equal(captured.options.headers.Authorization, "Bearer access-token");
  assert.deepEqual(JSON.parse(captured.options.body), {
    originId: "ST-A1", destination: "COR-A", poolMode: "corridor",
    pickupLatitude: 12.97, pickupLongitude: 77.59, dropoffLatitude: 12.98, dropoffLongitude: 77.6,
  });
});

test("fare quote submits map coordinates and never submits rider-estimated distance", async () => {
  let captured;
  const client = new ApiClient("http://api.test/api/v1", async (url, options) => {
    captured = { url, body: JSON.parse(options.body) };
    return new Response(JSON.stringify({ success: true, data: { distanceMeters: 3200 } }), { status: 200 });
  });
  await client.quoteFare({
    originId: "majestic", poolMode: "corridor",
    pickupLatitude: 12.97, pickupLongitude: 77.59, dropoffLatitude: 12.98, dropoffLongitude: 77.6,
    distanceMeters: 1,
  });
  assert.equal(captured.url, "http://api.test/api/v1/fare/quote");
  assert.deepEqual(captured.body, {
    poolMode: "corridor",
    pickupLatitude: 12.97, pickupLongitude: 77.59, dropoffLatitude: 12.98, dropoffLongitude: 77.6,
  });
});

test("transit itinerary requests use the authenticated backend and encode schedule parameters", async () => {
  let captured;
  const client = new ApiClient("https://api.test/api/v1", async (url, options) => {
    captured = { url, options };
    return new Response(JSON.stringify({ success: true, data: { dataSource: "Google Routes API" } }), { status: 200 });
  });
  client.setToken("rider-token");
  await client.transitRoute({
    originId: "majestic",
    destinationId: "indiranagar",
    departureTime: "2026-04-04T10:00:00.000Z",
  });
  assert.equal(captured.url, "https://api.test/api/v1/transit/route?originId=majestic&destinationId=indiranagar&departureTime=2026-04-04T10%3A00%3A00.000Z");
  assert.equal(captured.options.headers.Authorization, "Bearer rider-token");
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

test("role-specific authentication sends the selected account type", async () => {
  const requests = [];
  const client = new ApiClient("http://api.test/api/v1", async (url, options) => {
    requests.push({ url, options });
    return new Response(JSON.stringify({ success: true, data: {} }), { status: 201 });
  });
  await client.login({ email: "admin@example.test", password: "password", role: "admin" });
  await client.register({
    name: "Driver", email: "driver@example.test", phone: "", password: "a-long-password",
    role: "driver", invitationCode: "single-use-code",
  });
  assert.deepEqual(JSON.parse(requests[0].options.body), {
    email: "admin@example.test", password: "password", role: "admin",
  });
  assert.deepEqual(JSON.parse(requests[1].options.body), {
    name: "Driver", email: "driver@example.test", phone: "", password: "a-long-password",
    role: "driver", invitationCode: "single-use-code",
  });
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

  test("driver operations and subscription APIs use their authenticated and public routes", async () => {
    const requests = [];
    const client = new ApiClient("http://api.test/api/v1", async (url, options) => {
      requests.push({ url, options });
      return new Response(JSON.stringify({ success: true, data: {} }), { status: 200 });
    });
    client.setToken("session-token");
    await client.tripLocation("trip-1");
    await client.navigationRoute({ startLatitude: 12.9, startLongitude: 77.5, endLatitude: 13, endLongitude: 77.6 });
    await client.startPhoneVerification("+919876543210");
    await client.checkPhoneVerification("+919876543210", "123456");
    await client.updateDriverLocation({ latitude: 12.9, longitude: 77.5 });
    await client.driverDocuments();
    await client.driverEarnings();
    await client.driverRatingSummary();
    await client.assignBay("trip-1");
    await client.markBayArrival("trip-1");
    await client.adminBays();
    await client.subscriptionPlans();
    await client.adminSubscriptionPlans();
    await client.currentSubscription();
    await client.subscribe("commuter", "subscription-1");
    await client.cancelSubscription("subscription-1");
    assert.deepEqual(requests.map(({ url, options }) => [url, options.method || "GET"]), [
      ["http://api.test/api/v1/trips/trip-1/location", "GET"],
      ["http://api.test/api/v1/navigation/route", "POST"],
      ["http://api.test/api/v1/auth/otp/start", "POST"],
      ["http://api.test/api/v1/auth/otp/check", "POST"],
      ["http://api.test/api/v1/drivers/me/location", "POST"],
      ["http://api.test/api/v1/drivers/me/documents", "GET"],
      ["http://api.test/api/v1/drivers/me/earnings", "GET"],
      ["http://api.test/api/v1/drivers/me/ratings", "GET"],
      ["http://api.test/api/v1/drivers/me/trips/trip-1/bay", "POST"],
      ["http://api.test/api/v1/drivers/me/trips/trip-1/bay-arrival", "PATCH"],
      ["http://api.test/api/v1/admin/bays", "GET"],
      ["http://api.test/api/v1/subscriptions/plans", "GET"],
      ["http://api.test/api/v1/admin/subscription-plans", "GET"],
      ["http://api.test/api/v1/subscriptions/me", "GET"],
      ["http://api.test/api/v1/subscriptions", "POST"],
      ["http://api.test/api/v1/subscriptions/subscription-1", "DELETE"],
    ]);
    assert.equal(requests[11].options.headers.Authorization, undefined);
    assert.ok(requests.filter((_, index) => index !== 11).every(({ options }) => options.headers.Authorization === "Bearer session-token"));
    assert.equal(requests[14].options.headers["Idempotency-Key"], "subscription-1");
  });

  test("document uploads preserve FormData and downloads send the active bearer token", async () => {
    let captured;
    const client = new ApiClient("http://api.test/api/v1", async (url, options) => {
      captured = { url, options };
      return new Response(new Blob(["document"]), { status: 200 });
    });
    client.setToken("document-token");
    const form = new FormData();
    form.append("documentType", "license");
    form.append("file", new Blob(["license"]), "license.pdf");
    await client.uploadDriverDocument(form);
    assert.equal(captured.url, "http://api.test/api/v1/drivers/me/documents");
    assert.equal(captured.options.body, form);
    assert.equal(captured.options.headers["Content-Type"], undefined);
    assert.equal(captured.options.headers.Authorization, "Bearer document-token");
    const blob = await client.downloadDriverDocument("document-1");
    assert.equal(await blob.text(), "document");
    assert.equal(captured.url, "http://api.test/api/v1/admin/driver-documents/document-1/download");
    assert.equal(captured.options.headers.Authorization, "Bearer document-token");
  });

  test("document downloads fail closed when no session token is available", async () => {
    const client = new ApiClient("http://api.test/api/v1", async () => {
      throw new Error("fetch should not be called");
    });
    await assert.rejects(client.downloadDriverDocument("document-1"), (error) => {
      assert.ok(error instanceof ApiError);
      assert.equal(error.status, 401);
      return true;
    });
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

test("station and public metric reads are public while driver availability is authenticated", async () => {
  const requests = [];
  const client = new ApiClient("http://api.test/api/v1", async (url, options) => {
    requests.push({ url, options });
    return new Response(JSON.stringify({ success: true, data: [] }), { status: 200 });
  });
  client.setToken("driver-token");
  await client.stations();
  await client.publicMetrics();
  await client.driverProfile();
  await client.updateDriverAvailability("available");
  assert.deepEqual(requests.map(({ url, options }) => [url, options.method || "GET", options.headers.Authorization]), [
    ["http://api.test/api/v1/stations", "GET", undefined],
    ["http://api.test/api/v1/public/metrics", "GET", undefined],
    ["http://api.test/api/v1/drivers/me", "GET", "Bearer driver-token"],
    ["http://api.test/api/v1/drivers/me/availability", "PATCH", "Bearer driver-token"],
  ]);
  assert.deepEqual(JSON.parse(requests[3].options.body), { status: "available" });
});

test("admin wallet credit sends the audited amount and idempotency key", async () => {
  let captured;
  const client = new ApiClient("http://api.test/api/v1", async (url, options) => {
    captured = { url, options };
    return new Response(JSON.stringify({ success: true, data: { balancePaise: 5000 } }), { status: 201 });
  });
  client.setToken("admin-token");
  const result = await client.creditWallet("rider-1", { amountPaise: 2500, description: "Service adjustment" }, "credit-1");
  assert.equal(result.balancePaise, 5000);
  assert.equal(captured.url, "http://api.test/api/v1/admin/wallets/rider-1/credits");
  assert.equal(captured.options.headers.Authorization, "Bearer admin-token");
  assert.equal(captured.options.headers["Idempotency-Key"], "credit-1");
  assert.deepEqual(JSON.parse(captured.options.body), { amountPaise: 2500, description: "Service adjustment" });
});

test("admin invitation creation uses the protected API route", async () => {
  let captured;
  const client = new ApiClient("http://api.test/api/v1", async (url, options) => {
    captured = { url, options };
    return new Response(JSON.stringify({ success: true, data: { role: "driver", invitationCode: "once" } }), { status: 201 });
  });
  client.setToken("admin-token");
  await client.createInvitation("driver");
  assert.equal(captured.url, "http://api.test/api/v1/admin/invitations");
  assert.match(captured.options.headers.Authorization, /^Bearer /);
  assert.deepEqual(JSON.parse(captured.options.body), { role: "driver" });
});

test("support, trip rating, fleet, and dispatch methods call their role APIs", async () => {
  const requests = [];
  const client = new ApiClient("http://api.test/api/v1", async (url, options) => {
    requests.push({ url, options });
    return new Response(JSON.stringify({ success: true, data: [] }), { status: 200 });
  });
  client.setToken("session-token");
  await client.supportTickets();
  await client.createSupportTicket({ category: "trip", subject: "Pickup", message: "I need help." });
  await client.rateTrip("trip-1", { stars: 5, comment: "Good ride" });
  await client.adminSupportTickets();
  await client.updateSupportTicketStatus("ticket-1", "resolved");
  await client.assignDriverVehicle("driver-1", "vehicle-1");
  await client.vehicles();
  await client.dispatchPending();
  assert.deepEqual(requests.map(({ url, options }) => [url, options.method || "GET"]), [
    ["http://api.test/api/v1/support/tickets", "GET"],
    ["http://api.test/api/v1/support/tickets", "POST"],
    ["http://api.test/api/v1/trips/trip-1/rating", "POST"],
    ["http://api.test/api/v1/admin/support/tickets", "GET"],
    ["http://api.test/api/v1/admin/support/tickets/ticket-1/status", "PATCH"],
    ["http://api.test/api/v1/admin/drivers/driver-1/vehicle", "PATCH"],
    ["http://api.test/api/v1/vehicles", "GET"],
    ["http://api.test/api/v1/admin/dispatch/run", "POST"],
  ]);
  assert.ok(requests.every(({ options }) => options.headers.Authorization?.startsWith("Bearer ")));
});