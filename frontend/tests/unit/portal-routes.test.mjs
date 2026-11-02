import test from "node:test";
import assert from "node:assert/strict";
import { AUTH_PAGES, PORTAL_PAGES, homeForRole, parsePortalRoute, pathFor } from "../../scripts/core/portal-routes.mjs";

test("portal catalog covers the requested authentication and role pages", () => {
  assert.deepEqual(Object.keys(PORTAL_PAGES).sort(), ["admin", "driver", "rider"]);
  assert.deepEqual(PORTAL_PAGES.rider.map(({ id }) => id), ["dashboard", "transit", "destination-select", "fare-preview", "matching", "live-trip", "payment", "payment-success", "trip-history", "wallet", "profile", "rating", "support", "subscriptions"]);
  assert.deepEqual(PORTAL_PAGES.driver.map(({ id }) => id), ["dashboard", "availability", "assignment", "navigation", "bay-arrival", "otp-check", "trip-summary", "earnings", "ratings", "profile", "verification"]);
  assert.deepEqual(PORTAL_PAGES.admin.map(({ id }) => id), ["dashboard", "analytics", "trips", "pools", "drivers", "users", "stations", "bays", "vehicles", "incidents", "subscriptions", "reports"]);
  assert.deepEqual(AUTH_PAGES.map(({ id }) => id), ["welcome", "login-rider", "register-rider", "login-driver", "register-driver", "login-admin", "register-admin", "consent", "onboarding", "otp-verification"]);
  for (const [role, pages] of Object.entries(PORTAL_PAGES)) {
    for (const { id } of pages) {
      const route = parsePortalRoute(pathFor(role, id));
      assert.equal(route?.area, role);
      assert.equal(route?.page, id);
    }
  }
});

test("portal routes retain parameters and reject unknown destinations", () => {
  const route = parsePortalRoute("#/rider/live-trip?tripId=trip-1");
  assert.equal(route.area, "rider");
  assert.equal(route.page, "live-trip");
  assert.equal(route.params.get("tripId"), "trip-1");
  assert.equal(parsePortalRoute("#/admin/not-a-page"), null);
  assert.equal(parsePortalRoute("#/__proto__/dashboard"), null);
  assert.equal(pathFor("admin", "trips"), "#/admin/trips");
});

test("role home routes point to each dashboard", () => {
  assert.equal(homeForRole("rider"), "#/rider/dashboard");
  assert.equal(homeForRole("driver"), "#/driver/dashboard");
  assert.equal(homeForRole("admin"), "#/admin/dashboard");
  assert.equal(homeForRole("unknown"), "#/auth/welcome");
  assert.equal(homeForRole("__proto__"), "#/auth/welcome");
});

test("authentication routes separate each workspace and registration path", () => {
  for (const role of ["rider", "driver", "admin"]) {
    assert.equal(parsePortalRoute(pathFor("auth", `login-${role}`))?.page, `login-${role}`);
    assert.equal(parsePortalRoute(pathFor("auth", `register-${role}`))?.page, `register-${role}`);
  }
});