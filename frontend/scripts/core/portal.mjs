import { ApiClient, ApiError } from "./api.mjs";
import { AUTH_PAGES, PORTAL_PAGES, homeForRole, parsePortalRoute, pathFor } from "./portal-routes.mjs";
import { clearRouteMap, clearRouteMaps, createBookingMap, showLocationMap, showPoolRouteMap, showRouteMap } from "./route-map.mjs";

const apiOverride = new URLSearchParams(location.search).get("api");
const apiBase = window.NIMMA_API_BASE || apiOverride || `${location.protocol}//${location.hostname || "localhost"}:8080/api/v1`;
const api = new ApiClient(apiBase);
const state = {
  user: null,
  pendingRoute: "",
  tripInput: null,
  stations: [],
  quote: null,
  trips: [],
  paidTripIds: new Set(),
  selectedTripId: "",
  lastPayment: null,
  pendingMessage: "",
  renderId: 0,
  locationWatchId: null,
  locationPollId: null,
  lastLocation: null,
  pendingPhone: "",
  bookingMap: null,
  bookingPoints: { pickup: null, dropoff: null },
  bookingSelection: "pickup",
};
const byId = (id) => document.getElementById(id);
const money = (paise, currency = "INR") => new Intl.NumberFormat("en-IN", { style: "currency", currency }).format((Number(paise) || 0) / 100);
const localDateTimeValue = (date) => {
  const offsetDate = new Date(date.getTime() - date.getTimezoneOffset() * 60000);
  return offsetDate.toISOString().slice(0, 16);
};

function escapeHtml(value) {
  return String(value ?? "").replace(/[&<>"']/g, (character) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[character]);
}

function showAuthMessage(message, type = "") {
  const region = byId("authMessage");
  region.textContent = message;
  region.className = `form-message${type ? ` is-${type}` : ""}`;
}

function showMessage(message, type = "") {
  const region = byId("appMessage");
  region.textContent = message;
  region.className = `app-message is-visible${type ? ` is-${type}` : ""}`;
}

function stopLocationMonitoring() {
  if (state.locationWatchId !== null && navigator.geolocation) navigator.geolocation.clearWatch(state.locationWatchId);
  if (state.locationPollId !== null) clearInterval(state.locationPollId);
  state.locationWatchId = null;
  state.locationPollId = null;
}

function updateBookingPoint(type, point) {
  state.bookingPoints[type] = point;
  state.quote = null;
  state.tripInput = null;
  byId(type === "pickup" ? "pickupPointSummary" : "dropoffPointSummary").textContent = point
    ? `${type === "pickup" ? "Pickup" : "Drop-off"}: ${point.latitude.toFixed(5)}, ${point.longitude.toFixed(5)}`
    : `${type === "pickup" ? "Pickup" : "Drop-off"}: not selected`;
  byId("bookingQuoteOutput").textContent = point
    ? "Calculate a fresh route and fare after selecting both map pins."
    : "Set both map pins to calculate a route and quote.";
  byId("bookingCreateTrip").disabled = true;
  byId("bookingCreateTrip").classList.add("hidden");
}

function renderPoolRoutePreview(section, output, preview) {
  const stops = Array.isArray(preview.stops) ? preview.stops : [];
  if (!stops.length) {
    section.classList.remove("hidden");
    output.textContent = "The shared trip has no drop-off points to display.";
    return;
  }
  section.classList.remove("hidden");
  output.innerHTML = `<dl class="definition-list"><div><dt>Your direct route</dt><dd>${(preview.directDistanceMeters / 1000).toFixed(2)} km · ${Math.ceil(preview.directDurationSeconds / 60)} min</dd></div><div><dt>Your route to your drop-off in pool order</dt><dd>${(preview.sharedDistanceMeters / 1000).toFixed(2)} km · ${Math.ceil(preview.sharedDurationSeconds / 60)} min</dd></div><div><dt>Extra distance for your ride</dt><dd>+${(preview.distanceDeviationMeters / 1000).toFixed(2)} km</dd></div><div><dt>Estimated extra travel time</dt><dd>+${Math.ceil(preview.timeWastedSeconds / 60)} min</dd></div><div><dt>Entire pooled route</dt><dd>${(preview.poolRouteDistanceMeters / 1000).toFixed(2)} km · ${Math.ceil(preview.poolRouteDurationSeconds / 60)} min</dd></div></dl><div id="poolRouteMap" class="route-map" role="region" aria-label="Shared route with anonymous passenger drop-off markers"></div><p class="muted">Stop numbers show pool order only. Other passengers are not identified.</p>`;
  showPoolRouteMap(byId("poolRouteMap"), preview.geometry, stops);
}

function renderTransitRoute(container, route) {
  const legs = Array.isArray(route.transitLegs) ? route.transitLegs : [];
  container.innerHTML = `<p class="eyebrow">Schedule estimate · ${escapeHtml(route.dataSource)}</p><p><strong>${new Date(route.departureTime).toLocaleString()}</strong> departure · <strong>${new Date(route.arrivalTime).toLocaleString()}</strong> arrival · ${Math.ceil(route.durationSeconds / 60)} min total</p><ol class="transit-leg-list">${legs.map((leg) => `<li><strong>${escapeHtml(leg.lineShortName || leg.lineName || leg.vehicleType || "Transit service")}</strong> · ${escapeHtml(leg.departureStop)} at ${new Date(leg.departureTime).toLocaleTimeString()} → ${escapeHtml(leg.arrivalStop)} at ${new Date(leg.arrivalTime).toLocaleTimeString()}${leg.headsign ? ` · towards ${escapeHtml(leg.headsign)}` : ""}</li>`).join("")}</ol><p class="muted">Fetched ${new Date(route.retrievedAt).toLocaleTimeString()}. These are Google Maps itinerary estimates, not guaranteed live arrivals; walking or bus transfers may be included.</p>`;
}

function phoneVerificationContent(user) {
  const phone = user.phone || "";
  const status = user.phoneVerified ? `Verified phone: ${escapeHtml(phone)}` : "Verify your phone number to support account recovery and driver operations.";
  return `<section class="panel"><div class="section-head"><h2>Phone verification</h2><span class="status-pill">${user.phoneVerified ? "Verified" : "Not verified"}</span></div><p class="muted">${status}</p>${user.phoneVerified ? "" : `<label>Phone number (E.164)<input id="verificationPhone" type="tel" value="${escapeHtml(phone)}" placeholder="+919876543210" autocomplete="tel"></label><button class="button button-secondary" type="button" data-action="send-phone-code">Send SMS code</button><div id="phoneCodePanel" class="hidden"><label>SMS code<input id="verificationCode" inputmode="numeric" autocomplete="one-time-code" maxlength="10"></label><button class="button button-primary" type="button" data-action="verify-phone-code">Verify phone</button></div><p class="muted">SMS is sent only after you request it. Your phone number is sent to the configured SMS provider.</p>`}</section>`;
}

function setAuthContent(route) {
  const content = byId("authContent");
  const welcome = route.page === "welcome";
  const login = route.page.startsWith("login-");
  const register = route.page.startsWith("register-");
  const role = login || register ? route.page.slice(route.page.lastIndexOf("-") + 1) : "";
  const roleLabel = role === "admin" ? "Operations administrator" : `${role.charAt(0).toUpperCase()}${role.slice(1)}`;
  const definition = route.definition || AUTH_PAGES[0];
  byId("authHeading").innerHTML = welcome ? "One metro exit.<br><em>Every journey connected.</em>" : escapeHtml(definition.title);
  if (welcome) {
    content.innerHTML = `<p class="eyebrow">Account access</p><h2>Choose your workspace</h2><p class="muted">Each workspace has its own sign-in and account creation flow. Driver and administrator registration requires a single-use invitation.</p><div class="link-grid">${["rider", "driver", "admin"].map((item) => `<article class="link-tile"><strong>${item === "admin" ? "Operations admin" : `${item[0].toUpperCase()}${item.slice(1)}`}</strong><span>${item === "rider" ? "Plan rides, review trips, and manage your wallet." : item === "driver" ? "Manage availability and assigned trips." : "Monitor service operations and manage access."}</span><div class="button-row"><a class="button button-primary button-small" href="${pathFor("auth", `login-${item}`)}">Sign in</a><a class="button button-quiet button-small" href="${pathFor("auth", `register-${item}`)}">Create account</a></div></article>`).join("")}</div><div class="auth-links"><a href="${pathFor("auth", "consent")}">Privacy and consent</a><a href="${pathFor("auth", "otp-verification")}">Verification details</a></div>`;
  } else if (login) {
    content.innerHTML = `<div class="tabs"><a class="tab is-active" href="${pathFor("auth", `login-${role}`)}">Sign in</a><a class="tab" href="${pathFor("auth", `register-${role}`)}">Create account</a></div><form id="loginForm" class="form-stack" data-role="${role}"><div><h2>${escapeHtml(roleLabel)} sign in</h2><p class="muted">${escapeHtml(definition.description)}</p></div><label>Email address<input name="email" type="email" autocomplete="email" required maxlength="254"></label><label>Password<input name="password" type="password" autocomplete="current-password" required maxlength="72"></label><button class="button button-primary" type="submit">Sign in</button></form><div class="auth-links"><a href="${pathFor("auth", `register-${role}`)}">Create a ${escapeHtml(role)} account</a><a href="${pathFor("auth", "welcome")}">Choose another workspace</a></div>`;
  } else if (register) {
    const invitation = role !== "rider" ? `<label>Single-use invitation code<input name="invitationCode" type="text" autocomplete="off" required minlength="32" maxlength="128"></label>` : "";
    const phone = role !== "admin" ? `<label>Phone (optional)<input name="phone" type="tel" autocomplete="tel" maxlength="40"></label>` : "";
    const verification = role === "driver" ? `<p class="muted">New driver accounts are unverified and remain offline until an administrator approves them.</p>` : "";
    content.innerHTML = `<div class="tabs"><a class="tab" href="${pathFor("auth", `login-${role}`)}">Sign in</a><a class="tab is-active" href="${pathFor("auth", `register-${role}`)}">Create account</a></div><form id="registerForm" class="form-stack" data-role="${role}"><div><h2>${escapeHtml(definition.title)}</h2><p class="muted">${escapeHtml(definition.description)} Passwords require at least 12 characters.</p></div><label>Full name<input name="name" type="text" autocomplete="name" required maxlength="100"></label><label>Email address<input name="email" type="email" autocomplete="email" required maxlength="254"></label>${phone}${invitation}${verification}<label>Password<input name="password" type="password" autocomplete="new-password" required minlength="12" maxlength="72"></label><label class="consent-check"><input name="consent" type="checkbox" required><span>I agree to the service terms described in <a class="text-link" href="${pathFor("auth", "consent")}">Privacy and consent</a>.</span></label><button class="button button-primary" type="submit">Create ${escapeHtml(role)} account</button></form><div class="auth-links"><a href="${pathFor("auth", `login-${role}`)}">Already registered? Sign in</a><a href="${pathFor("auth", "welcome")}">Choose another workspace</a></div>`;
  } else {
    const verification = route.page === "otp-verification";
    const text = verification
      ? "Phone verification is available from the profile page when an SMS provider is configured. The account is not treated as verified until the provider confirms the code."
      : route.page === "consent"
        ? "The service uses your device location only after you explicitly start trip tracking. Phone numbers are sent to the configured SMS provider only when you request a verification code. Driver documents are stored privately for administrator review."
        : "New rider, driver, and admin accounts use separate pages. Driver and admin account creation requires an invitation issued from an authenticated admin workspace.";
    content.innerHTML = `<p class="eyebrow">${escapeHtml(definition.label)}</p><h2>${escapeHtml(definition.title)}</h2><p class="muted">${escapeHtml(text)}</p><div class="button-row"><a class="button button-primary" href="${pathFor("auth", "register-rider")}">Create rider account</a><a class="button button-secondary" href="${pathFor("auth", "login-rider")}">Rider sign in</a></div><div class="auth-links"><a href="${pathFor("auth", "welcome")}">Choose another workspace</a></div>`;
  }
  showAuthMessage("");
}

function routeLinks(area, currentPage) {
  return PORTAL_PAGES[area].map((page) => `<a class="nav-button${page.id === currentPage ? " is-current" : ""}" href="${pathFor(area, page.id)}"${page.id === currentPage ? ' aria-current="page"' : ""}><span aria-hidden="true">${page.id === currentPage ? "●" : "○"}</span>${escapeHtml(page.label)}</a>`).join("");
}

function heading(route, actions = "") {
  return `<div class="page-heading"><div><p class="eyebrow">${escapeHtml(route.area)} workspace</p><h1>${escapeHtml(route.definition.title)}</h1><p>${escapeHtml(route.definition.description)}</p></div>${actions ? `<div class="page-heading-actions">${actions}</div>` : ""}</div>`;
}

function actionButton(action, label, value = "", style = "button-secondary") {
  const data = value ? ` data-value="${escapeHtml(value)}"` : "";
  return `<button class="button ${style}" type="button" data-action="${action}"${data}>${escapeHtml(label)}</button>`;
}

function routeTile(area, page) {
  return `<a class="link-tile" href="${pathFor(area, page.id)}"><strong>${escapeHtml(page.label)}</strong><span>${escapeHtml(page.description)}</span></a>`;
}

function notConnected(title, detail) {
  return `<section class="panel"><p class="eyebrow">Not connected</p><h2>${escapeHtml(title)}</h2><p class="muted">${escapeHtml(detail)}</p><div class="callout"><span class="note-icon" aria-hidden="true">i</span><span>No operational data or successful action is simulated here. This screen becomes active when the corresponding backend endpoint is available.</span></div></section>`;
}

function riderContent(route) {
  const page = route.page;
  if (page === "dashboard") {
    return `${heading(route, `<a class="button button-primary" href="${pathFor("rider", "destination-select")}">Plan a trip</a>`)}<div class="metric-grid"><article class="metric"><span>Trips on account</span><strong id="riderTripCount">—</strong></article><article class="metric"><span>Wallet balance</span><strong id="riderBalance">—</strong></article><article class="metric"><span>Account role</span><strong>${escapeHtml(state.user.role)}</strong></article><article class="metric"><span>Service</span><strong>Connected</strong></article></div><div class="section-head"><h2>Rider pages</h2><a class="text-link" href="${pathFor("rider", "trip-history")}">Trip history</a></div><div class="link-grid">${["transit", "destination-select", "fare-preview", "matching", "live-trip", "payment", "trip-history", "wallet", "profile", "rating", "support", "subscriptions"].map((id) => routeTile("rider", PORTAL_PAGES.rider.find((item) => item.id === id))).join("")}</div>`;
  }
  if (page === "destination-select") {
    return `${heading(route)}<section class="panel"><form id="tripForm" class="form-stack"><label>Nearest Namma Metro station<select id="originStationSelect" name="originId" required><option value="">Loading Namma Metro stations…</option></select><small>Used for metro pickup-bay allocation. Set the exact pickup on the map.</small></label><div class="form-grid"><label>Ride mode<select name="poolMode"><option value="corridor">Corridor pool</option><option value="solo">Solo</option></select></label><label>Passengers<input name="partySize" type="number" min="1" max="8" value="1" required></label></div><label class="consent-check"><input name="requiresAccessible" type="checkbox"><span>Request a wheelchair-accessible pickup bay</span></label><div class="section-head"><div><h2>Choose pickup and destination</h2><p class="muted">Select a pin mode, then click the map. Route distance and ETA are calculated by the backend road router.</p></div><button class="button button-quiet button-small" type="button" data-action="use-current-location">Use my location for pickup</button></div><div class="button-row"><button class="button button-secondary button-small is-current" type="button" data-action="select-map-point" data-value="pickup" aria-pressed="true">Set pickup pin</button><button class="button button-quiet button-small" type="button" data-action="select-map-point" data-value="dropoff" aria-pressed="false">Set drop-off pin</button></div><div id="bookingMap" class="route-map booking-map" role="application" aria-label="Interactive map to select your pickup and drop-off"></div><div class="split-grid"><p id="pickupPointSummary" class="muted">Pickup: not selected</p><p id="dropoffPointSummary" class="muted">Drop-off: not selected</p></div><p class="muted">OpenStreetMap tiles reveal the approximate map area viewed to the tile provider. Device location is only used if you request it.</p><button class="button button-primary" type="submit">Calculate route and fare</button><div id="bookingQuoteOutput" class="empty-state" aria-live="polite">Set both map pins to calculate a route and quote.</div><button id="bookingCreateTrip" class="button button-green hidden" type="button" data-action="create-trip" disabled>Request this trip</button></form></section>`;
  }
  if (page === "transit") {
    return `${heading(route)}<section class="panel"><form id="transitRouteForm" class="form-stack"><div><h2>Find a train or metro connection</h2><p class="muted">Choose two Namma Metro stations and a departure time. The server requests a fresh Google Maps transit itinerary.</p></div><div class="form-grid"><label>From<select id="transitOriginSelect" name="originId" required><option value="">Loading stations…</option></select></label><label>To<select id="transitDestinationSelect" name="destinationId" required><option value="">Loading stations…</option></select></label></div><label>Depart at<input name="departureTime" type="datetime-local" value="${localDateTimeValue(new Date())}" required></label><button class="button button-primary" type="submit">Find transit times</button></form><div id="transitRouteOutput" class="empty-state" aria-live="polite">Transit itineraries are estimates and depend on Google Maps coverage for the selected time and location.</div></section>`;
  }
  if (page === "fare-preview") {
    return `${heading(route)}<section class="panel" id="quotePanel"><p class="empty-state">${state.tripInput ? "Calculating quote from the backend…" : `Choose journey details first. <a class="text-link" href="${pathFor("rider", "destination-select")}">Plan a trip</a>`}</p></section>`;
  }
  if (page === "matching") return `${heading(route, actionButton("refresh", "Refresh candidates"))}<section class="panel"><div id="matchingOutput" class="empty-state">Candidate matches are returned by the API for your active trip. Matching does not create a pool or assign a vehicle.</div></section>`;
  if (page === "live-trip") return `${heading(route, actionButton("refresh", "Refresh details"))}<section class="panel"><div id="liveTripOutput" class="empty-state">Select a trip from your history to view its latest server status.</div><section id="poolRouteSection" class="panel hidden"><h2>Shared route and drop-offs</h2><p class="muted">The map shows drop-off points only. Passenger names, account IDs, and other personal details are never shown.</p><div id="poolRouteOutput" class="empty-state">Loading shared route…</div></section><section id="tripLocationPanel" class="panel"><h2>Live driver location</h2><p class="muted">Location appears only while the driver shares it on an active trip.</p><div id="tripLocationOutput" class="empty-state">Waiting for an active trip.</div></section><a class="text-link" href="${pathFor("rider", "trip-history")}">Open trip history</a></section>`;
  if (page === "payment") return `${heading(route)}<section class="panel"><div id="paymentOutput" class="empty-state">Choose an eligible trip from your history before paying from your wallet.</div><a class="text-link" href="${pathFor("rider", "trip-history")}">Choose a trip</a></section>`;
  if (page === "payment-success") return `${heading(route)}<section class="panel"><div id="paymentReceipt" class="empty-state">No payment has been made in this session.</div><a class="text-link" href="${pathFor("rider", "trip-history")}">Return to trips</a></section>`;
  if (page === "trip-history") return `${heading(route, actionButton("refresh", "Refresh trips"))}<section id="tripHistoryList" class="empty-state">Loading trips…</section>`;
  if (page === "wallet") return `${heading(route, actionButton("refresh", "Refresh wallet"))}<section id="walletContent" class="panel"><p class="empty-state">Loading wallet…</p></section>`;
  if (page === "profile") return `${heading(route)}<section class="panel"><dl class="definition-list"><div><dt>Name</dt><dd>${escapeHtml(state.user.name)}</dd></div><div><dt>Email</dt><dd>${escapeHtml(state.user.email)}</dd></div><div><dt>Account ID</dt><dd>${escapeHtml(state.user.id)}</dd></div><div><dt>Role</dt><dd>${escapeHtml(state.user.role)}</dd></div></dl></section>${phoneVerificationContent(state.user)}`;
  if (page === "rating") return `${heading(route)}<section class="panel"><form id="ratingForm" class="form-stack"><label>Completed trip<select name="tripId" id="ratingTripSelect" required><option value="">Loading completed trips…</option></select></label><label>Rating<select name="stars" required><option value="5">5 — Excellent</option><option value="4">4 — Good</option><option value="3">3 — Okay</option><option value="2">2 — Poor</option><option value="1">1 — Very poor</option></select></label><label>Comment (optional)<textarea name="comment" rows="4" maxlength="1000"></textarea></label><button class="button button-primary" type="submit">Submit rating</button></form></section>`;
  if (page === "support") return `${heading(route)}<form id="supportForm" class="panel form-stack"><div><h2>Contact support</h2><p class="muted">Share the issue and the support team will update its status here.</p></div><label>Category<select name="category" required><option value="trip">Trip</option><option value="wallet">Wallet or payment</option><option value="safety">Safety</option><option value="other">Other</option></select></label><label>Subject<input name="subject" required maxlength="120"></label><label>Details<textarea name="message" rows="5" required maxlength="2000"></textarea></label><button class="button button-primary" type="submit">Submit support request</button></form><section class="panel"><div class="section-head"><h2>Your requests</h2></div><div id="supportTicketList" class="empty-state">Loading requests…</div></section>`;
  if (page === "subscriptions") return `${heading(route)}<section class="panel"><div class="section-head"><h2>Your current plan</h2><span id="currentSubscriptionStatus" class="status-pill">Loading</span></div><div id="currentSubscriptionContent" class="empty-state">Loading subscription…</div></section><section class="panel"><h2>Available plans</h2><p class="muted">Plan prices are charged once for 30 days from your wallet. Discount limits count non-cancelled rides created during the current calendar month.</p><div id="subscriptionPlanList" class="empty-state">Loading plans…</div></section>`;
}

function adminContent(route) {
  const page = route.page;
  if (page === "dashboard") {
    const shortcuts = PORTAL_PAGES.admin.filter((item) => item.id !== "dashboard").map((item) => routeTile("admin", item)).join("");
    return `${heading(route, actionButton("refresh", "Refresh data"))}<div class="metric-grid" id="adminMetricGrid"><article class="metric"><span>Trips</span><strong>Loading</strong></article></div><div class="section-head"><h2>Operations pages</h2></div><div class="link-grid">${shortcuts}</div>`;
  }
  if (page === "analytics") return `${heading(route, `<button class="button button-secondary" data-action="run-simulation" type="button">Run simulation</button><button class="button button-quiet" data-action="refresh" type="button">Refresh metrics</button>`)}<div class="metric-grid" id="adminMetricGrid"><article class="metric"><span>Metrics</span><strong>Loading</strong></article></div><section class="panel"><div class="section-head"><h2>Simulation result</h2></div><pre id="simulationOutput" class="data-output">Run a deterministic scenario to load results.</pre></section>`;
  if (page === "pools") return `${heading(route, `<button class="button button-primary" data-action="run-dispatch" type="button">Form pools and dispatch trips</button>`)}<section class="panel"><p class="muted">Processes eligible corridor requests into pools and dispatches eligible solo trips using verified drivers and vehicles.</p><pre id="dispatchOutput" class="data-output">Run dispatch to process queued requests.</pre></section>`;
  if (page === "trips") return `${heading(route, actionButton("refresh", "Refresh trips"))}<section id="adminTripList" class="empty-state">Loading trips…</section><section id="adminTripDetail" class="panel hidden"></section>`;
  if (page === "drivers") return `${heading(route, actionButton("refresh", "Refresh drivers"))}<section id="adminDriverList" class="empty-state">Loading directory…</section><section class="panel"><div class="section-head"><h2>Driver verification documents</h2></div><div id="adminDriverDocuments" class="empty-state">Loading documents…</div></section>`;
  if (page === "vehicles") return `${heading(route, actionButton("refresh", "Refresh vehicles"))}<section id="adminVehicleList" class="empty-state">Loading fleet…</section>`;
  if (page === "stations") return `${heading(route, actionButton("refresh", "Refresh stations"))}<section id="adminStationList" class="empty-state">Loading Namma Metro station catalog…</section>`;
  if (page === "bays") return `${heading(route)}<form id="bayForm" class="panel form-stack"><h2>Add or update a station bay</h2><div class="form-grid"><label>Station<select name="stationId" id="bayStationSelect" required><option value="">Loading stations…</option></select></label><label>Bay label<input name="label" required maxlength="80" placeholder="Pickup bay A"></label><label>Vehicle capacity<input name="capacity" type="number" min="1" max="100" value="4" required></label><label>Walk distance (meters)<input name="walkingMeters" type="number" min="0" max="500" step="1" value="0" required></label><label>Base queue length<input name="queueLength" type="number" min="0" max="1000" value="0" required></label></div><label class="consent-check"><input name="accessible" type="checkbox"><span>Wheelchair accessible</span></label><label class="consent-check"><input name="open" type="checkbox" checked><span>Open for assignment</span></label><button class="button button-primary" type="submit">Save bay</button></form><section id="adminBayList" class="empty-state">Loading bays…</section>`;
  if (page === "subscriptions") return `${heading(route)}<form id="subscriptionPlanForm" class="panel form-stack"><h2>Create or update a ride plan</h2><div class="form-grid"><label>Plan ID<input name="id" required maxlength="40" pattern="[a-z0-9-]+" placeholder="commuter-20"></label><label>Plan name<input name="name" required maxlength="100"></label><label>Price (rupees)<input name="priceRupees" type="number" min="0" step="0.01" required></label><label>Monthly ride limit<input name="monthlyRideLimit" type="number" min="0" value="10" required><small>Zero means unlimited rides for the active discount.</small></label><label>Discount (%)<input name="discountPercent" type="number" min="0" max="100" step="0.01" required></label></div><label class="consent-check"><input name="active" type="checkbox" checked><span>Available to riders</span></label><button class="button button-primary" type="submit">Save plan</button></form><section id="adminSubscriptionPlanList" class="empty-state">Loading subscription plans…</section>`;
  if (page === "users") return `${heading(route, actionButton("refresh", "Refresh directory"))}<div class="split-grid"><section class="panel"><div class="section-head"><h2>Riders</h2></div><div id="adminRiderList" class="empty-state">Loading…</div></section><section class="panel"><div class="section-head"><h2>Drivers</h2></div><div id="adminUserDriverList" class="empty-state">Loading…</div></section></div><form id="portalInviteForm" class="panel form-stack"><div><p class="eyebrow">Protected access</p><h2>Invite a driver or administrator</h2><p class="muted">Creates a single-use invitation that expires in seven days. The code is shown once; share it only with the intended person.</p></div><label>Account type<select name="role" required><option value="driver">Driver</option><option value="admin">Administrator</option></select></label><button class="button button-secondary" type="submit">Create invitation</button><output id="invitationResult" class="data-output hidden" aria-live="polite"></output></form><form id="portalCreditForm" class="panel form-stack"><div><p class="eyebrow">Audited adjustment</p><h2>Credit rider wallet</h2><p class="muted">This writes an audited ledger credit in rupees. It does not charge a payment method.</p></div><div class="form-grid"><label>Rider user ID<input name="userId" required autocomplete="off"></label><label>Amount in rupees<input name="amountRupees" type="number" min="0.01" step="0.01" required></label></div><label>Description<input name="description" maxlength="240" required></label><button class="button button-primary" type="submit">Create credit entry</button></form>`;
  if (page === "incidents") return `${heading(route, actionButton("refresh", "Refresh tickets"))}<section id="adminSupportTicketList" class="empty-state">Loading support tickets…</section>`;
  if (page === "reports") return `${heading(route, `<button class="button button-primary" data-action="export-trips" type="button">Download trips CSV</button>`)}<section class="panel"><h2>Trip report</h2><p class="muted">Exports the latest trip records available to this admin account, including statuses and INR fares.</p></section>`;
  return `${heading(route)}${notConnected(`${route.definition.title} data is unavailable`, route.definition.description)}`;
}

function driverContent(route) {
  if (route.page === "dashboard") {
    const shortcuts = PORTAL_PAGES.driver.filter((item) => item.id !== "dashboard").map((item) => routeTile("driver", item)).join("");
    return `${heading(route)}<div class="metric-grid"><article class="metric"><span>Driver status</span><strong id="driverStatusValue">Loading</strong></article><article class="metric"><span>Assigned trips</span><strong id="driverTripCount">—</strong></article><article class="metric"><span>Vehicle capacity</span><strong id="driverCapacity">—</strong></article><article class="metric"><span>Vehicle assignment</span><strong id="driverVehicle">—</strong></article></div><section id="driverStatusPanel" class="panel">Loading driver profile…</section><div class="section-head"><h2>Driver pages</h2></div><div class="link-grid">${shortcuts}</div>`;
  }
  if (route.page === "availability") return `${heading(route)}<section id="driverStatusPanel" class="panel">Loading availability…</section>`;
  if (route.page === "assignment") return `${heading(route, actionButton("refresh", "Refresh assignments"))}<section id="driverAssignments" class="empty-state">Loading assigned trips…</section>`;
  if (route.page === "trip-summary") return `${heading(route)}<section id="driverTripSummary" class="panel"><p class="empty-state">Select a trip from assignments to view its summary.</p><a class="text-link" href="${pathFor("driver", "assignment")}">Open assignments</a></section>`;
  if (route.page === "profile") return `${heading(route)}<section id="driverProfilePanel" class="panel">Loading driver profile…</section>`;
  if (route.page === "navigation") return `${heading(route)}<section class="panel"><p class="muted">Location sharing is off until you start it. Browser permission is required. The server stores your latest position only for active assigned trips; riders on those trips can see it.</p><div class="form-grid"><label>Active trip<select id="navigationTripSelect" required><option value="">Choose an active trip</option></select></label><button class="button button-primary" type="button" data-action="start-location-sharing">Start location sharing</button><button class="button button-quiet" type="button" data-action="stop-location-sharing">Stop sharing</button></div><p id="driverLocationStatus" class="form-message" aria-live="polite">Location sharing is stopped.</p></section><form id="navigationForm" class="panel form-stack"><h2>Build a road route</h2><p class="muted">The backend gets road directions from configured OSRM. The map uses OpenStreetMap tiles; its servers receive the approximate map area requested by your browser.</p><div class="form-grid"><label>Destination latitude<input name="endLatitude" type="number" min="-90" max="90" step="any" required></label><label>Destination longitude<input name="endLongitude" type="number" min="-180" max="180" step="any" required></label></div><button class="button button-secondary" type="submit">Calculate route from current location</button><div id="navigationRouteOutput" class="empty-state">Start location sharing to get your current position.</div></form>`;
  if (route.page === "bay-arrival") return `${heading(route, actionButton("refresh", "Refresh trips"))}<section id="driverBayTrips" class="empty-state">Loading assigned trips…</section>`;
  if (route.page === "earnings") return `${heading(route, actionButton("refresh", "Refresh earnings"))}<section id="driverEarningsPanel" class="panel">Loading completed-trip earnings…</section>`;
  if (route.page === "ratings") return `${heading(route, actionButton("refresh", "Refresh ratings"))}<section id="driverRatingPanel" class="panel">Loading rider feedback summary…</section>`;
  if (route.page === "verification") return `${heading(route)}<form id="driverDocumentForm" class="panel form-stack"><h2>Submit verification document</h2><p class="muted">PDF, JPEG or PNG; maximum 10 MB. Files are stored outside the web root and only administrators can download them.</p><label>Document type<select name="documentType" required><option value="driving_license">Driving licence</option><option value="identity">Identity document</option><option value="vehicle_registration">Vehicle registration</option><option value="insurance">Vehicle insurance</option></select></label><label>Document file<input name="document" type="file" accept=".pdf,.jpg,.jpeg,.png,application/pdf,image/jpeg,image/png" required></label><button class="button button-primary" type="submit">Upload for review</button></form><section class="panel"><div class="section-head"><h2>Submitted documents</h2></div><div id="driverDocumentList" class="empty-state">Loading documents…</div></section>${phoneVerificationContent(state.user)}`;
  return `${heading(route)}${notConnected(`${route.definition.title} is not connected`, route.definition.description)}`;
}

function renderWorkspace(route) {
  stopLocationMonitoring();
  clearRouteMaps();
  document.body.dataset.role = route.area;
  byId("authView").classList.add("hidden");
  byId("workspaceView").classList.remove("hidden");
  byId("workspaceRole").textContent = route.area.charAt(0).toUpperCase() + route.area.slice(1);
  byId("workspaceNav").innerHTML = routeLinks(route.area, route.page);
  byId("apiOriginLabel").textContent = apiBase;
  byId("pageContent").innerHTML = route.area === "rider" ? riderContent(route) : route.area === "admin" ? adminContent(route) : driverContent(route);
  byId("appMessage").className = "app-message";
  if (state.pendingMessage) {
    showMessage(state.pendingMessage, "error");
    state.pendingMessage = "";
  }
  document.title = `Nimma Saarthi | ${route.definition.title}`;
  const renderId = ++state.renderId;
  loadPageData(route, renderId).catch((error) => {
    if (renderId === state.renderId) showMessage(error.message || "The request could not be completed.", "error");
  });
}

function renderAuth(route) {
  clearRouteMaps();
  delete document.body.dataset.role;
  byId("workspaceView").classList.add("hidden");
  byId("authView").classList.remove("hidden");
  document.title = `Nimma Saarthi | ${route.definition.title}`;
  setAuthContent(route);
}

function navigate(hash) {
  location.hash = hash;
}

function render() {
  const route = parsePortalRoute(location.hash);
  if (!route) {
    navigate(pathFor("auth", "welcome"));
    return;
  }
  if (route.area === "auth") {
    if (state.user) {
      navigate(homeForRole(state.user.role));
      return;
    }
    renderAuth(route);
    return;
  }
  if (!state.user) {
    state.pendingRoute = location.hash;
    navigate(pathFor("auth", `login-${route.area}`));
    return;
  }
  if (state.user.role !== route.area) {
    state.pendingMessage = `This account does not have access to the ${route.area} workspace.`;
    navigate(homeForRole(state.user.role));
    return;
  }
  renderWorkspace(route);
}

function activateSession(session) {
  if (!session?.accessToken || !session?.user?.id || !Object.hasOwn(PORTAL_PAGES, session.user.role)) throw new Error("This account returned an unsupported role or an incomplete session.");
  api.setToken(session.accessToken);
  state.user = session.user;
  byId("identityLabel").textContent = `${session.user.name} · ${session.user.role}`;
  byId("signOutButton").classList.remove("hidden");
  const intended = parsePortalRoute(state.pendingRoute);
  state.pendingRoute = "";
  navigate(intended?.area === session.user.role ? `#/${intended.area}/${intended.page}${intended.params.toString() ? `?${intended.params}` : ""}` : homeForRole(session.user.role));
}

function clearSession() {
  api.clearSession();
  state.user = null;
  state.tripInput = null;
  state.quote = null;
  state.trips = [];
  state.selectedTripId = "";
  state.lastPayment = null;
  byId("identityLabel").textContent = "Signed out";
  byId("signOutButton").classList.add("hidden");
  navigate(pathFor("auth", "welcome"));
}

function renderTrips(container, trips, admin = false) {
  if (!Array.isArray(trips) || !trips.length) {
    container.innerHTML = `<p class="empty-state">No trips are available for this account.</p>`;
    return;
  }
  container.innerHTML = `<div class="data-table-wrap"><table class="data-table"><thead><tr><th>Route</th><th>Status</th><th>Mode</th><th>Fare</th><th>Actions</th></tr></thead><tbody>${trips.map((trip) => {
    const paid = state.paidTripIds.has(trip.id);
    const riderActions = !admin && trip.status === "requested" ? `<button class="button button-secondary button-small" data-action="cancel-trip" data-value="${escapeHtml(trip.id)}" type="button">Cancel</button>${!paid && trip.farePaise > 0 ? `<button class="button button-primary button-small" data-action="pay-trip" data-value="${escapeHtml(trip.id)}" type="button">Pay</button>` : paid ? `<span class="status-pill">Paid</span>` : ""}` : "";
    return `<tr><td><strong>${escapeHtml(trip.originId)} → ${escapeHtml(trip.destination)}</strong><small>${escapeHtml(trip.id)}</small></td><td><span class="status-pill">${escapeHtml(trip.status)}</span></td><td>${escapeHtml(trip.poolMode)}</td><td>${money(trip.farePaise)}</td><td><div class="button-row"><button class="button button-quiet button-small" data-action="${admin ? "admin-trip-detail" : "live-trip"}" data-value="${escapeHtml(trip.id)}" type="button">Details</button>${riderActions}</div></td></tr>`;
  }).join("")}</tbody></table></div>`;
}

function renderRecords(container, records) {
  if (!Array.isArray(records) || !records.length) {
    container.innerHTML = `<p class="empty-state">No records returned.</p>`;
    return;
  }
  container.innerHTML = `<div class="data-table-wrap"><table class="data-table"><thead><tr><th>Name</th><th>ID</th><th>Status / role</th></tr></thead><tbody>${records.map((record) => `<tr><td>${escapeHtml(record.name || "Unnamed")}</td><td>${escapeHtml(record.id)}</td><td>${escapeHtml(record.status || record.role || "—")}</td></tr>`).join("")}</tbody></table></div>`;
}

function renderSupportTickets(container, tickets, admin = false) {
  if (!Array.isArray(tickets) || !tickets.length) {
    container.innerHTML = `<p class="empty-state">${admin ? "No support tickets have been submitted." : "You have not submitted any support requests."}</p>`;
    return;
  }
  container.innerHTML = `<div class="data-table-wrap"><table class="data-table"><thead><tr>${admin ? "<th>Rider</th>" : ""}<th>Category</th><th>Subject</th><th>Details</th><th>Status</th>${admin ? "<th>Update</th>" : ""}</tr></thead><tbody>${tickets.map((ticket) => `<tr>${admin ? `<td>${escapeHtml(ticket.userId)}</td>` : ""}<td>${escapeHtml(ticket.category)}</td><td>${escapeHtml(ticket.subject)}</td><td>${escapeHtml(ticket.message)}</td><td><span class="status-pill">${escapeHtml(ticket.status)}</span></td>${admin ? `<td><div class="button-row"><select aria-label="Update status for ticket ${escapeHtml(ticket.id)}"><option value="open"${ticket.status === "open" ? " selected" : ""}>Open</option><option value="in_progress"${ticket.status === "in_progress" ? " selected" : ""}>In progress</option><option value="resolved"${ticket.status === "resolved" ? " selected" : ""}>Resolved</option></select><button class="button button-secondary button-small" type="button" data-action="support-ticket-status" data-value="${escapeHtml(ticket.id)}">Save</button></div></td>` : ""}</tr>`).join("")}</tbody></table></div>`;
}

function renderDrivers(container, drivers, vehicles = []) {
  if (!Array.isArray(drivers) || !drivers.length) {
    container.innerHTML = `<p class="empty-state">No driver accounts returned.</p>`;
    return;
  }
  const assignableVehicles = vehicles.filter((vehicle) => vehicle.active && vehicle.verified);
  container.innerHTML = `<div class="data-table-wrap"><table class="data-table"><thead><tr><th>Name</th><th>ID</th><th>Status</th><th>Verification</th><th>Vehicle</th><th>Actions</th></tr></thead><tbody>${drivers.map((driver) => `<tr><td>${escapeHtml(driver.name)}</td><td>${escapeHtml(driver.id)}</td><td>${escapeHtml(driver.status)}</td><td>${driver.verified ? "Verified" : "Pending"}</td><td><select aria-label="Vehicle for driver ${escapeHtml(driver.name)}" data-driver-vehicle><option value="">Select a vehicle</option>${assignableVehicles.map((vehicle) => `<option value="${escapeHtml(vehicle.id)}"${driver.vehicleId === vehicle.id ? " selected" : ""}>${escapeHtml(vehicle.label)} · ${escapeHtml(vehicle.vehicleType)}</option>`).join("")}</select></td><td><div class="button-row"><button class="button button-secondary button-small" type="button" data-action="verify-driver" data-value="${escapeHtml(driver.id)}" data-verified="${!driver.verified}">${driver.verified ? "Revoke" : "Approve"}</button><button class="button button-quiet button-small" type="button" data-action="assign-driver-vehicle" data-value="${escapeHtml(driver.id)}"${assignableVehicles.length ? "" : " disabled"}>Assign</button></div></td></tr>`).join("")}</tbody></table></div>`;
}

function renderVehicles(container, vehicles) {
  if (!Array.isArray(vehicles) || !vehicles.length) {
    container.innerHTML = `<p class="empty-state">No fleet vehicles returned.</p>`;
    return;
  }
  container.innerHTML = `<div class="data-table-wrap"><table class="data-table"><thead><tr><th>Label</th><th>Type</th><th>Capacity</th><th>Driver</th><th>Active</th><th>Verified</th><th>Accessibility</th></tr></thead><tbody>${vehicles.map((vehicle) => `<tr><td>${escapeHtml(vehicle.label)}</td><td>${escapeHtml(vehicle.vehicleType)}</td><td>${vehicle.capacity}</td><td>${escapeHtml(vehicle.driverStatus)}</td><td>${vehicle.active ? "Yes" : "No"}</td><td>${vehicle.verified ? "Yes" : "No"}</td><td>${vehicle.accessibilitySupported ? "Supported" : "—"}</td></tr>`).join("")}</tbody></table></div>`;
}

function renderStations(container, stations) {
  if (!Array.isArray(stations) || !stations.length) {
    container.innerHTML = `<p class="empty-state">No active stations were returned by the database catalog.</p>`;
    return;
  }
  container.innerHTML = `<div class="data-table-wrap"><table class="data-table"><thead><tr><th>Station</th><th>Line</th><th>Station ID</th></tr></thead><tbody>${stations.map((station) => `<tr><td>${escapeHtml(station.name)}</td><td>${escapeHtml((station.lines || []).map((line) => `${line.charAt(0).toUpperCase()}${line.slice(1)} Line`).join(" / "))}</td><td>${escapeHtml(station.id)}</td></tr>`).join("")}</tbody></table></div>`;
}

function renderMetrics(metrics) {
  const container = byId("adminMetricGrid");
  if (!container) return;
  const labels = {
    activeRiders: "Active riders",
    ridersTotal: "Registered riders",
    activeDrivers: "Available drivers",
    driversTotal: "Registered drivers",
    activeVehicles: "Active vehicles",
    stationsTotal: "Metro stations",
    tripsTotal: "Total trips",
    tripsRequested: "Requested trips",
    tripsCompleted: "Completed trips",
    activePools: "Active pools",
    poolTripRate: "Pooled trip rate",
  };
  const entries = Object.entries(labels).map(([key, label]) => [label, metrics?.[key]]);
  container.innerHTML = entries.map(([label, value]) => `<article class="metric"><span>${escapeHtml(label)}</span><strong>${value === undefined ? "—" : escapeHtml(typeof value === "number" && label === "Pool trip rate" ? `${(value * 100).toFixed(1)}%` : value)}</strong></article>`).join("");
}

function renderDriverProfile(profile, { trips = [] } = {}) {
  const statusControl = !profile.verified
    ? `<p class="eyebrow">Verification</p><h2>Pending approval</h2><p class="muted">Your account is offline until operations verifies your driver profile.</p>`
    : profile.status === "busy"
    ? `<p class="eyebrow">Availability</p><h2>On trip</h2><p class="muted">Availability is locked while a trip is in progress.</p>`
    : `<p class="eyebrow">Availability</p><h2>${escapeHtml(profile.status)}</h2><p class="muted">Set whether operations may consider you for new assignments.</p><div class="button-row">${profile.status === "available" ? actionButton("driver-availability", "Go offline", "offline", "button-quiet") : actionButton("driver-availability", "Go available", "available", "button-green")}</div>`;
  const dashboardStatus = byId("driverStatusPanel");
  if (dashboardStatus) dashboardStatus.innerHTML = statusControl;
  const profilePanel = byId("driverProfilePanel");
  if (profilePanel) profilePanel.innerHTML = `<dl class="definition-list"><div><dt>Name</dt><dd>${escapeHtml(profile.name)}</dd></div><div><dt>Driver ID</dt><dd>${escapeHtml(profile.id)}</dd></div><div><dt>Vehicle ID</dt><dd>${escapeHtml(profile.vehicleId || "Not assigned")}</dd></div><div><dt>Vehicle capacity</dt><dd>${profile.capacity || "Not provided"}</dd></div><div><dt>Availability</dt><dd>${escapeHtml(profile.status)}</dd></div><div><dt>Verification</dt><dd>${profile.verified ? "Verified" : "Pending approval"}</dd></div></dl>`;
  if (byId("driverStatusValue")) byId("driverStatusValue").textContent = profile.status;
  if (byId("driverCapacity")) byId("driverCapacity").textContent = profile.capacity ? String(profile.capacity) : "—";
  if (byId("driverVehicle")) byId("driverVehicle").textContent = profile.vehicleId ? "Assigned" : "Unassigned";
  if (byId("driverTripCount")) byId("driverTripCount").textContent = String(trips.length);
}

function renderDriverTrips(container, trips) {
  if (!Array.isArray(trips) || !trips.length) {
    container.innerHTML = `<p class="empty-state">No trips are currently assigned to this driver account.</p>`;
    return;
  }
  container.innerHTML = `<div class="data-table-wrap"><table class="data-table"><thead><tr><th>Route</th><th>Status</th><th>Mode</th><th>Fare</th><th>Action</th></tr></thead><tbody>${trips.map((trip) => {
    const next = trip.status === "matched" ? "in_progress" : trip.status === "in_progress" ? "completed" : "";
    const label = next === "in_progress" ? "Start trip" : next === "completed" ? "Complete trip" : "No action";
    const bayActions = ["matched", "in_progress"].includes(trip.status) ? `<button class="button button-secondary button-small" type="button" data-action="assign-bay" data-value="${escapeHtml(trip.id)}">Assign pickup bay</button><button class="button button-quiet button-small" type="button" data-action="mark-bay-arrival" data-value="${escapeHtml(trip.id)}">Mark bay arrival</button>` : "";
    return `<tr><td><strong>${escapeHtml(trip.originId)} → ${escapeHtml(trip.destination)}</strong><small>${escapeHtml(trip.id)}</small></td><td><span class="status-pill">${escapeHtml(trip.status)}</span></td><td>${escapeHtml(trip.poolMode)}</td><td>${money(trip.farePaise)}</td><td><div class="button-row"><a class="button button-quiet button-small" href="${pathFor("driver", "trip-summary", `tripId=${encodeURIComponent(trip.id)}`)}">Details</a>${next ? `<button class="button button-green button-small" type="button" data-action="driver-trip-status" data-value="${escapeHtml(trip.id)}" data-status="${next}">${label}</button>` : `<span class="muted">${label}</span>`}${bayActions}</div></td></tr>`;
  }).join("")}</tbody></table></div>`;
}

function renderDriverDocuments(container, documents, admin = false) {
  if (!Array.isArray(documents) || !documents.length) {
    container.innerHTML = `<p class="empty-state">${admin ? "No driver documents are awaiting review." : "No documents have been uploaded."}</p>`;
    return;
  }
  container.innerHTML = `<div class="data-table-wrap"><table class="data-table"><thead><tr>${admin ? "<th>Driver</th>" : ""}<th>Type</th><th>File</th><th>Uploaded</th><th>Status</th>${admin ? "<th>Review</th>" : ""}</tr></thead><tbody>${documents.map((document) => `<tr>${admin ? `<td>${escapeHtml(document.driverId)}</td>` : ""}<td>${escapeHtml(document.documentType.replaceAll("_", " "))}</td><td>${admin ? `<button class="button button-quiet button-small" type="button" data-action="download-driver-document" data-value="${escapeHtml(document.id)}">${escapeHtml(document.fileName)}</button>` : escapeHtml(document.fileName)}</td><td>${new Date(document.uploadedAt).toLocaleString()}</td><td><span class="status-pill">${escapeHtml(document.status)}</span>${document.reviewNote ? `<small>${escapeHtml(document.reviewNote)}</small>` : ""}</td>${admin ? `<td><div class="button-row"><button class="button button-secondary button-small" type="button" data-action="review-driver-document" data-value="${escapeHtml(document.id)}" data-status="approved">Approve</button><button class="button button-quiet button-small" type="button" data-action="review-driver-document" data-value="${escapeHtml(document.id)}" data-status="rejected">Reject</button></div></td>` : ""}</tr>`).join("")}</tbody></table></div>`;
}

function renderAdminBays(container, bays) {
  if (!Array.isArray(bays) || !bays.length) {
    container.innerHTML = `<p class="empty-state">No pickup bays configured. Add a bay above.</p>`;
    return;
  }
  container.innerHTML = `<div class="data-table-wrap"><table class="data-table"><thead><tr><th>Station</th><th>Bay</th><th>Capacity</th><th>Queue</th><th>Walk</th><th>Accessible</th><th>Status</th><th>Action</th></tr></thead><tbody>${bays.map((bay) => `<tr><td>${escapeHtml(bay.stationId)}</td><td>${escapeHtml(bay.label)}</td><td>${bay.capacity}</td><td>${bay.queueLength}</td><td>${bay.walkingMeters} m</td><td>${bay.accessible ? "Yes" : "No"}</td><td>${bay.open ? "Open" : "Closed"}</td><td><button class="button button-quiet button-small" type="button" data-action="toggle-bay" data-value="${escapeHtml(bay.id)}" data-open="${!bay.open}">${bay.open ? "Close" : "Open"}</button></td></tr>`).join("")}</tbody></table></div>`;
}

function renderSubscriptionPlans(container, plans, currentSubscription = null, admin = false) {
  if (!Array.isArray(plans) || !plans.length) {
    container.innerHTML = `<p class="empty-state">No subscription plans are available.</p>`;
    return;
  }
  container.innerHTML = `<div class="link-grid">${plans.map((plan) => `<article class="link-tile"><strong>${escapeHtml(plan.name)}</strong><span>${money(plan.pricePaise)} · ${plan.monthlyRideLimit ? `${plan.monthlyRideLimit} rides/month` : "Unlimited rides"} · ${(plan.discountBps / 100).toFixed(2)}% off</span>${admin ? `<span>${plan.active ? "Available to riders" : "Inactive"}</span><button class="button button-secondary button-small" type="button" data-action="toggle-subscription-plan" data-value="${escapeHtml(plan.id)}" data-active="${!plan.active}">${plan.active ? "Deactivate" : "Activate"}</button>` : currentSubscription ? `<span>Cancel your current plan before choosing another.</span>` : `<button class="button button-primary button-small" type="button" data-action="subscribe" data-value="${escapeHtml(plan.id)}">Subscribe from wallet</button>`}</article>`).join("")}</div>`;
}

function renderSubscription(container, statusContainer, subscription) {
  if (!subscription) {
    statusContainer.textContent = "No active plan";
    container.innerHTML = `<p class="empty-state">You do not have an active subscription.</p>`;
    return;
  }
  statusContainer.textContent = "Active";
  container.innerHTML = `<p><strong>${escapeHtml(subscription.planName)}</strong></p><p class="muted">Valid through ${new Date(subscription.endsAt).toLocaleString()}.</p><button class="button button-quiet button-small" type="button" data-action="cancel-subscription" data-value="${escapeHtml(subscription.id)}">Cancel renewal</button><p class="muted">Cancellation ends the plan immediately. No refund or recurring charge is issued.</p>`;
}

function renderWallet(wallet) {
  const container = byId("walletContent");
  if (!container) return;
  const entries = Array.isArray(wallet.entries) ? wallet.entries : [];
  container.innerHTML = `<div class="wallet-balance"><span>Available balance</span><strong>${money(wallet.balancePaise, wallet.currency || "INR")}</strong><small>${escapeHtml(wallet.currency || "INR")}</small></div><div class="section-head"><h2>Ledger</h2></div>${entries.length ? `<div class="ledger-list">${entries.map((entry) => `<div class="ledger-row"><div><strong>${escapeHtml(entry.description)}</strong><small>${escapeHtml(entry.entryType)} · ${new Date(entry.createdAt).toLocaleString()}</small></div><strong class="${entry.amountPaise >= 0 ? "positive" : "negative"}">${money(entry.amountPaise, wallet.currency || "INR")}</strong></div>`).join("")}</div>` : `<p class="empty-state">No ledger entries yet.</p>`}<p class="muted">Card processing and self-service wallet funding are not connected.</p>`;
}

async function refreshTripLocation(tripId, renderId) {
  try {
    const location = await api.tripLocation(tripId);
    if (renderId !== state.renderId) return;
    clearRouteMap(byId("tripLocationMap"));
    byId("tripLocationOutput").innerHTML = `<p>Live position · GPS accuracy ±${Math.round(location.accuracyMeters)} m · updated ${new Date(location.recordedAt).toLocaleTimeString()}.</p><div id="tripLocationMap" class="route-map" role="region" aria-label="Map showing the live driver location"></div><p class="muted">Map tiles from OpenStreetMap may reveal the approximate map area your browser views.</p>`;
    showLocationMap(byId("tripLocationMap"), Number(location.latitude), Number(location.longitude));
  } catch (error) {
    if (renderId !== state.renderId) return;
    if (error instanceof ApiError && error.status === 404) {
      byId("tripLocationOutput").textContent = "The driver has not shared a location for this active trip.";
      return;
    }
    throw error;
  }
}

async function loadPageData(route, renderId) {
  if (route.area === "rider") {
    if (route.page === "dashboard") {
      const [trips, wallet] = await Promise.all([api.listTrips(), api.wallet()]);
      if (renderId !== state.renderId) return;
      state.trips = trips;
      state.paidTripIds = new Set((wallet.entries || []).filter((entry) => entry.entryType === "debit" && entry.tripId).map((entry) => entry.tripId));
      byId("riderTripCount").textContent = String(trips.length);
      byId("riderBalance").textContent = money(wallet.balancePaise, wallet.currency || "INR");
    } else if (route.page === "destination-select") {
      state.stations = await api.stations();
      if (renderId === state.renderId) {
        const select = byId("originStationSelect");
        select.replaceChildren(new Option("Choose a station", ""));
        state.stations.forEach((station) => {
          const option = new Option(`${station.name} · ${(station.lines || []).join(" / ")}`, station.id);
          select.add(option);
        });
        if (!state.stations.length) showMessage("No active metro stations are in the database catalog.", "error");
        state.bookingPoints = { pickup: null, dropoff: null };
        state.bookingSelection = "pickup";
        state.bookingMap = createBookingMap(byId("bookingMap"),
          (point) => updateBookingPoint("pickup", point),
          (point) => updateBookingPoint("dropoff", point));
      }
    } else if (route.page === "transit") {
      state.stations = await api.stations();
      if (renderId === state.renderId) {
        for (const id of ["transitOriginSelect", "transitDestinationSelect"]) {
          const select = byId(id);
          select.replaceChildren(new Option("Choose a station", ""));
          state.stations.forEach((station) => select.add(new Option(station.name, station.id)));
        }
        if (state.stations.length < 2) {
          byId("transitRouteOutput").textContent = "At least two active metro stations are needed to plan a transit route.";
        }
      }
    } else if (route.page === "trip-history") {
      const [trips, wallet] = await Promise.all([api.listTrips(), api.wallet()]);
      state.trips = trips;
      state.paidTripIds = new Set((wallet.entries || []).filter((entry) => entry.entryType === "debit" && entry.tripId).map((entry) => entry.tripId));
      if (renderId === state.renderId) renderTrips(byId("tripHistoryList"), state.trips);
    } else if (route.page === "wallet") {
      const wallet = await api.wallet();
      if (renderId === state.renderId) renderWallet(wallet);
    } else if (route.page === "rating") {
      const trips = await api.listTrips();
      if (renderId === state.renderId) {
        state.trips = trips;
        const select = byId("ratingTripSelect");
        select.replaceChildren(new Option("Choose a completed trip", ""));
        trips.filter((trip) => trip.status === "completed").forEach((trip) => {
          select.add(new Option(`${trip.originId} → ${trip.destination} · ${new Date(trip.updatedAt).toLocaleDateString()}`, trip.id));
        });
        if (select.options.length === 1) select.options[0].textContent = "No completed trips are ready for a rating";
      }
    } else if (route.page === "support") {
      const tickets = await api.supportTickets();
      if (renderId === state.renderId) renderSupportTickets(byId("supportTicketList"), tickets);
    } else if (route.page === "subscriptions") {
      const plans = await api.subscriptionPlans();
      let subscription = null;
      try {
        subscription = await api.currentSubscription();
      } catch (error) {
        if (!(error instanceof ApiError) || error.status !== 404) throw error;
      }
      if (renderId === state.renderId) {
        renderSubscription(byId("currentSubscriptionContent"), byId("currentSubscriptionStatus"), subscription);
        renderSubscriptionPlans(byId("subscriptionPlanList"), plans, subscription);
      }
    } else if (route.page === "fare-preview" && state.tripInput) {
      state.quote = await api.quoteFare(state.tripInput);
      if (renderId === state.renderId) {
        byId("quotePanel").innerHTML = `<p class="eyebrow">Server quote</p><h2>${money(state.quote.totalPaise, state.quote.currency || "INR")}</h2><dl class="definition-list"><div><dt>Base fare</dt><dd>${money(state.quote.basePaise, state.quote.currency || "INR")}</dd></div><div><dt>Distance fare</dt><dd>${money(state.quote.distancePaise, state.quote.currency || "INR")}</dd></div><div><dt>Pool discount</dt><dd>${money(state.quote.discountPaise, state.quote.currency || "INR")}</dd></div>${state.quote.subscriptionDiscountPaise ? `<div><dt>Subscription discount</dt><dd>${money(state.quote.subscriptionDiscountPaise, state.quote.currency || "INR")}</dd></div>` : ""}</dl><p class="muted">Quote is an estimate, not a payment authorization.</p><div class="button-row"><button class="button button-primary" type="button" data-action="create-trip">Request trip</button><a class="button button-quiet" href="${pathFor("rider", "destination-select")}">Edit journey</a></div>`;
      }
    } else if (route.page === "matching") {
      const matches = await api.matchPool();
      if (renderId === state.renderId) byId("matchingOutput").innerHTML = `<pre class="data-output">${escapeHtml(JSON.stringify(matches, null, 2))}</pre>`;
    } else if (route.page === "live-trip") {
      const tripId = route.params.get("tripId") || state.selectedTripId;
      if (tripId) {
        state.selectedTripId = tripId;
        const trip = await api.getTrip(tripId);
        if (renderId === state.renderId) {
          byId("liveTripOutput").innerHTML = `<dl class="definition-list"><div><dt>Route</dt><dd>${escapeHtml(trip.originId)} → ${escapeHtml(trip.destination)}</dd></div><div><dt>Road distance</dt><dd>${(trip.distanceMeters / 1000).toFixed(2)} km</dd></div><div><dt>Status</dt><dd>${escapeHtml(trip.status)}</dd></div><div><dt>Ride mode</dt><dd>${escapeHtml(trip.poolMode)}</dd></div><div><dt>Fare</dt><dd>${money(trip.farePaise)}</dd></div><div><dt>Last updated</dt><dd>${escapeHtml(trip.updatedAt ? new Date(trip.updatedAt).toLocaleString() : "Not provided")}</dd></div></dl>`;
          if (trip.poolId && ["matched", "in_progress"].includes(trip.status)) {
            const preview = await api.poolRoutePreview(tripId);
            if (renderId === state.renderId) renderPoolRoutePreview(byId("poolRouteSection"), byId("poolRouteOutput"), preview);
          }
          if (["matched", "in_progress"].includes(trip.status)) {
            await refreshTripLocation(tripId, renderId);
            state.locationPollId = setInterval(() => {
              refreshTripLocation(tripId, renderId).catch((error) => showMessage(error.message, "error"));
            }, 10000);
          } else {
            byId("tripLocationOutput").textContent = "Live tracking is available while the trip is matched or in progress.";
          }
        }
      }
    } else if (route.page === "payment") {
      const tripId = route.params.get("tripId") || state.selectedTripId;
      if (tripId) {
        state.selectedTripId = tripId;
        const trip = await api.getTrip(tripId);
        if (renderId === state.renderId) byId("paymentOutput").innerHTML = `<p>Trip ${escapeHtml(trip.id)} · ${escapeHtml(trip.status)} · ${money(trip.farePaise)}</p><button class="button button-primary" type="button" data-action="confirm-payment" data-value="${escapeHtml(trip.id)}">Pay from wallet</button>`;
      }
    } else if (route.page === "payment-success" && state.lastPayment) {
      byId("paymentReceipt").textContent = `Payment recorded: ${money(state.lastPayment.amountPaise)}. Wallet balance: ${money(state.lastPayment.balancePaise)}.`;
    }
    return;
  }
  if (route.area === "admin") {
    if (["dashboard", "analytics"].includes(route.page)) {
      const metrics = await api.metrics();
      if (renderId === state.renderId) renderMetrics(metrics);
      if (route.page === "dashboard") {
        const trips = await api.listTrips();
        if (renderId === state.renderId) state.trips = trips;
      }
    } else if (route.page === "trips") {
      const trips = await api.listTrips();
      if (renderId === state.renderId) renderTrips(byId("adminTripList"), trips, true);
    } else if (route.page === "drivers") {
      const [drivers, vehicles, documents] = await Promise.all([api.drivers(), api.vehicles(), api.adminDriverDocuments()]);
      if (renderId === state.renderId) {
        renderDrivers(byId("adminDriverList"), drivers, vehicles);
        renderDriverDocuments(byId("adminDriverDocuments"), documents, true);
      }
    } else if (route.page === "vehicles") {
      const vehicles = await api.vehicles();
      if (renderId === state.renderId) renderVehicles(byId("adminVehicleList"), vehicles);
    } else if (route.page === "stations") {
      const stations = await api.stations();
      if (renderId === state.renderId) renderStations(byId("adminStationList"), stations);
    } else if (route.page === "users") {
      const [riders, drivers, vehicles] = await Promise.all([api.riders(), api.drivers(), api.vehicles()]);
      if (renderId === state.renderId) {
        renderRecords(byId("adminRiderList"), riders);
        renderDrivers(byId("adminUserDriverList"), drivers, vehicles);
      }
    } else if (route.page === "incidents") {
      const tickets = await api.adminSupportTickets();
      if (renderId === state.renderId) renderSupportTickets(byId("adminSupportTicketList"), tickets, true);
    } else if (route.page === "bays") {
      const [bays, stations] = await Promise.all([api.adminBays(), api.stations()]);
      if (renderId === state.renderId) {
        renderAdminBays(byId("adminBayList"), bays);
        const select = byId("bayStationSelect");
        select.replaceChildren(new Option("Choose a station", ""));
        stations.forEach((station) => select.add(new Option(station.name, station.id)));
      }
    } else if (route.page === "subscriptions") {
      const plans = await api.adminSubscriptionPlans();
      if (renderId === state.renderId) renderSubscriptionPlans(byId("adminSubscriptionPlanList"), plans, null, true);
    }
    return;
  }
  if (route.area === "driver") {
    const needsTrips = ["dashboard", "assignment", "trip-summary", "navigation", "bay-arrival"].includes(route.page);
    const [profile, trips] = await Promise.all([
      api.driverProfile(),
      needsTrips ? api.listTrips() : Promise.resolve([]),
    ]);
    if (renderId !== state.renderId) return;
    if (["dashboard", "availability", "profile"].includes(route.page)) renderDriverProfile(profile, { trips });
    if (route.page === "dashboard") state.trips = trips;
    if (route.page === "assignment") renderDriverTrips(byId("driverAssignments"), trips);
    if (route.page === "bay-arrival") renderDriverTrips(byId("driverBayTrips"), trips);
    if (route.page === "navigation") {
      const select = byId("navigationTripSelect");
      select.replaceChildren(new Option("Choose an active trip", ""));
      select.required = true;
      trips.filter((trip) => ["matched", "in_progress"].includes(trip.status)).forEach((trip) => {
        select.add(new Option(`${trip.originId} → ${trip.destination} · ${trip.status}`, trip.id));
      });
    }
    if (route.page === "trip-summary") {
      const tripId = route.params.get("tripId") || state.selectedTripId;
      const trip = trips.find((item) => item.id === tripId);
      if (trip) byId("driverTripSummary").innerHTML = `<dl class="definition-list"><div><dt>Route</dt><dd>${escapeHtml(trip.originId)} → ${escapeHtml(trip.destination)}</dd></div><div><dt>Status</dt><dd>${escapeHtml(trip.status)}</dd></div><div><dt>Fare</dt><dd>${money(trip.farePaise)}</dd></div><div><dt>Distance</dt><dd>${Number(trip.distanceMeters).toLocaleString()} m</dd></div></dl>`;
    }
    if (route.page === "earnings") {
      const earnings = await api.driverEarnings();
      if (renderId === state.renderId) byId("driverEarningsPanel").innerHTML = `<div class="metric-grid"><article class="metric"><span>Completed trips</span><strong>${earnings.tripsCompleted}</strong></article><article class="metric"><span>Gross fares</span><strong>${money(earnings.grossPaise)}</strong></article><article class="metric"><span>Platform commission (15%)</span><strong>${money(earnings.commissionPaise)}</strong></article><article class="metric"><span>Estimated driver share</span><strong>${money(earnings.netPaise)}</strong></article></div><p class="muted">These are estimates based on completed trip fares. No payout or bank-transfer provider is connected.</p>`;
    }
    if (route.page === "ratings") {
      const summary = await api.driverRatingSummary();
      if (renderId === state.renderId) byId("driverRatingPanel").innerHTML = `<div class="metric-grid"><article class="metric"><span>Average rating</span><strong>${summary.count ? `${Number(summary.average).toFixed(2)} / 5` : "No ratings yet"}</strong></article><article class="metric"><span>Rider feedback</span><strong>${summary.count}</strong></article></div>`;
    }
    if (route.page === "verification") {
      const documents = await api.driverDocuments();
      if (renderId === state.renderId) renderDriverDocuments(byId("driverDocumentList"), documents);
    }
  }
}

async function handleSubmit(event) {
  const form = event.target;
  if (!(form instanceof HTMLFormElement)) return;
  event.preventDefault();
  const data = new FormData(form);
  const submit = form.querySelector('[type="submit"]');
  if (submit) submit.disabled = true;
  try {
    if (form.id === "loginForm") {
      activateSession(await api.login({ email: data.get("email"), password: data.get("password"), role: form.dataset.role }));
    } else if (form.id === "registerForm") {
      if (!data.get("consent")) throw new Error("Consent is required to create an account.");
      activateSession(await api.register({ name: data.get("name"), email: data.get("email"), phone: data.get("phone"), password: data.get("password"), role: form.dataset.role, invitationCode: data.get("invitationCode") }));
    } else if (form.id === "supportForm") {
      await api.createSupportTicket({
        category: String(data.get("category")),
        subject: String(data.get("subject")).trim(),
        message: String(data.get("message")).trim(),
      });
      form.reset();
      renderSupportTickets(byId("supportTicketList"), await api.supportTickets());
      showMessage("Support request submitted.", "success");
    } else if (form.id === "ratingForm") {
      const rating = await api.rateTrip(String(data.get("tripId")), {
        stars: Number(data.get("stars")),
        comment: String(data.get("comment") || "").trim(),
      });
      form.reset();
      showMessage(`Thanks for rating this trip ${rating.stars} out of 5.`, "success");
    } else if (form.id === "driverDocumentForm") {
      await api.uploadDriverDocument(data);
      form.reset();
      renderDriverDocuments(byId("driverDocumentList"), await api.driverDocuments());
      showMessage("Document uploaded securely for administrator review.", "success");
    } else if (form.id === "bayForm") {
      await api.saveBay({
        stationId: String(data.get("stationId")),
        label: String(data.get("label")).trim(),
        capacity: Number(data.get("capacity")),
        walkingMeters: Number(data.get("walkingMeters")),
        queueLength: Number(data.get("queueLength")),
        accessible: data.has("accessible"),
        open: data.has("open"),
      });
      form.reset();
      const bays = await api.adminBays();
      renderAdminBays(byId("adminBayList"), bays);
      showMessage("Station bay saved.", "success");
    } else if (form.id === "subscriptionPlanForm") {
      await api.saveSubscriptionPlan({
        id: String(data.get("id")).trim(),
        name: String(data.get("name")).trim(),
        pricePaise: Math.round(Number(data.get("priceRupees")) * 100),
        monthlyRideLimit: Number(data.get("monthlyRideLimit")),
        discountBps: Math.round(Number(data.get("discountPercent")) * 100),
        active: data.has("active"),
      });
      form.reset();
      renderSubscriptionPlans(byId("adminSubscriptionPlanList"), await api.adminSubscriptionPlans(), null, true);
      showMessage("Subscription plan saved.", "success");
    } else if (form.id === "navigationForm") {
      if (!state.lastLocation) throw new Error("Start location sharing and wait for a GPS fix first.");
      const route = await api.navigationRoute({
        startLatitude: state.lastLocation.latitude,
        startLongitude: state.lastLocation.longitude,
        endLatitude: Number(data.get("endLatitude")),
        endLongitude: Number(data.get("endLongitude")),
      });
      const steps = (route.steps || []).map((step, index) => `<li>${index + 1}. ${escapeHtml(step.maneuver || "Continue")} ${escapeHtml(step.name || "unnamed road")} · ${Math.round(step.distanceMeters)} m</li>`).join("");
      clearRouteMaps();
      byId("navigationRouteOutput").innerHTML = `<p><strong>${(route.distanceMeters / 1000).toFixed(2)} km</strong> · ${Math.ceil(route.durationSeconds / 60)} min estimated</p><div id="driverRouteMap" class="route-map" role="region" aria-label="Map showing the route returned by the routing service"></div><ol>${steps || "<li>Follow the route shown on the map.</li>"}</ol><p class="muted">Map tiles from OpenStreetMap may reveal the approximate map area your browser views.</p>`;
      showRouteMap(byId("driverRouteMap"), route.geometry);
      showMessage("Road route calculated by the configured OSRM service.", "success");
    } else if (form.id === "tripForm") {
      const { pickup, dropoff } = state.bookingPoints;
      if (!pickup || !dropoff) throw new Error("Choose both pickup and drop-off pins on the map.");
      state.tripInput = {
        originId: String(data.get("originId")).trim(),
        destination: `${dropoff.latitude.toFixed(5)}, ${dropoff.longitude.toFixed(5)}`,
        poolMode: String(data.get("poolMode") || "corridor"),
        pickupLatitude: pickup.latitude,
        pickupLongitude: pickup.longitude,
        dropoffLatitude: dropoff.latitude,
        dropoffLongitude: dropoff.longitude,
        partySize: Number(data.get("partySize") || 1),
        requiresAccessible: data.has("requiresAccessible"),
      };
      state.quote = await api.quoteFare(state.tripInput);
      byId("bookingQuoteOutput").innerHTML = `<p class="eyebrow">Road route and fare · calculated by OSRM</p><p><strong>${(state.quote.distanceMeters / 1000).toFixed(2)} km</strong> · ${Math.ceil(state.quote.durationSeconds / 60)} min estimated travel time</p><p><strong>${money(state.quote.totalPaise, state.quote.currency || "INR")}</strong> · includes the server-side fare and applicable discounts</p><p class="muted">The final fare is recalculated from the road route when the trip request is saved.</p>`;
      state.bookingMap?.setRoute(state.quote.geometry);
      byId("bookingCreateTrip").disabled = false;
      byId("bookingCreateTrip").classList.remove("hidden");
      showMessage("Road distance, duration and fare calculated by the backend.", "success");
    } else if (form.id === "transitRouteForm") {
      const originId = String(data.get("originId"));
      const destinationId = String(data.get("destinationId"));
      if (originId === destinationId) throw new Error("Choose different origin and destination stations.");
      const departureTime = new Date(String(data.get("departureTime")));
      if (Number.isNaN(departureTime.getTime())) throw new Error("Choose a valid departure date and time.");
      const route = await api.transitRoute({ originId, destinationId, departureTime: departureTime.toISOString() });
      renderTransitRoute(byId("transitRouteOutput"), route);
      showMessage("Transit itinerary loaded from Google Maps.", "success");
    } else if (form.id === "portalCreditForm") {
      if (!form.dataset.idempotencyKey) {
        const key = globalThis.crypto?.randomUUID?.() || `${Date.now()}-${Math.random().toString(16).slice(2)}`;
        form.dataset.idempotencyKey = `admin-credit-${key}`;
      }
      const result = await api.creditWallet(data.get("userId"), {
        amountPaise: Math.round(Number(data.get("amountRupees")) * 100),
        description: String(data.get("description")).trim(),
      }, form.dataset.idempotencyKey);
      showMessage(`Wallet credited. New balance: ${money(result.balancePaise)}.`, "success");
      form.reset();
      delete form.dataset.idempotencyKey;
    } else if (form.id === "portalInviteForm") {
      const invitation = await api.createInvitation(String(data.get("role")));
      const output = byId("invitationResult");
      output.textContent = `${invitation.role} invitation code (expires ${new Date(invitation.expiresAt).toLocaleString()}): ${invitation.invitationCode}`;
      output.classList.remove("hidden");
      showMessage("Invitation created. Copy and share the code securely; it is only shown here once.", "success");
    }
  } catch (error) {
    if (form.id === "loginForm" || form.id === "registerForm") showAuthMessage(error.message, "error");
    else showMessage(error.message, "error");
  } finally {
    if (submit?.isConnected) submit.disabled = false;
  }
}

async function handleClick(event) {
  const button = event.target.closest("[data-action]");
  if (!button) return;
  const action = button.dataset.action;
  const value = button.dataset.value;
  button.disabled = true;
  try {
    if (action === "refresh") {
      render();
    } else if (action === "create-trip") {
      if (!state.tripInput || !state.quote) throw new Error("Request a fresh fare quote before creating a trip.");
      const trip = await api.createTrip(state.tripInput);
      state.selectedTripId = trip.id;
      showMessage("Trip request created.", "success");
      navigate(pathFor("rider", "trip-history"));
    } else if (action === "select-map-point") {
      state.bookingSelection = value;
      state.bookingMap?.selectPoint(value);
      document.querySelectorAll('[data-action="select-map-point"]').forEach((item) => {
        const selected = item.dataset.value === value;
        item.classList.toggle("is-current", selected);
        item.setAttribute("aria-pressed", String(selected));
      });
    } else if (action === "use-current-location") {
      if (!navigator.geolocation) throw new Error("This browser does not support device location.");
      const position = await new Promise((resolve, reject) => {
        navigator.geolocation.getCurrentPosition(resolve, reject, { enableHighAccuracy: true, timeout: 15000, maximumAge: 30000 });
      });
      const pickup = { latitude: position.coords.latitude, longitude: position.coords.longitude };
      state.bookingMap?.setPoint("pickup", pickup);
      state.bookingMap?.focusPoint(pickup);
    } else if (action === "find-matches") {
      const matches = await api.matchPool();
      byId("matchingOutput").innerHTML = `<pre class="data-output">${escapeHtml(JSON.stringify(matches, null, 2))}</pre>`;
    } else if (action === "live-trip") {
      state.selectedTripId = value;
      navigate(pathFor("rider", "live-trip", `tripId=${encodeURIComponent(value)}`));
    } else if (action === "admin-trip-detail") {
      const trip = await api.getTrip(value);
      const detail = byId("adminTripDetail");
      detail.innerHTML = `<div class="section-head"><h2>Trip detail</h2><span class="status-pill">${escapeHtml(trip.status)}</span></div><dl class="definition-list"><div><dt>Trip ID</dt><dd>${escapeHtml(trip.id)}</dd></div><div><dt>Rider ID</dt><dd>${escapeHtml(trip.riderId)}</dd></div><div><dt>Driver ID</dt><dd>${escapeHtml(trip.driverId || "Unassigned")}</dd></div><div><dt>Route</dt><dd>${escapeHtml(trip.originId)} → ${escapeHtml(trip.destination)}</dd></div><div><dt>Fare</dt><dd>${money(trip.farePaise)}</dd></div></dl>`;
      detail.classList.remove("hidden");
    } else if (action === "verify-driver") {
      await api.verifyDriver(value, button.dataset.verified === "true");
      showMessage(button.dataset.verified === "true" ? "Driver account approved." : "Driver verification revoked.", "success");
      render();
    } else if (action === "assign-driver-vehicle") {
      const vehicleId = button.closest("tr").querySelector("[data-driver-vehicle]").value;
      if (!vehicleId) throw new Error("Choose a verified vehicle first.");
      await api.assignDriverVehicle(value, vehicleId);
      showMessage("Vehicle assigned to driver.", "success");
      render();
    } else if (action === "send-phone-code") {
      state.pendingPhone = byId("verificationPhone").value.trim();
      await api.startPhoneVerification(state.pendingPhone);
      byId("phoneCodePanel").classList.remove("hidden");
      showMessage("Verification code sent. It expires according to the SMS provider's policy.", "success");
    } else if (action === "verify-phone-code") {
      const result = await api.checkPhoneVerification(state.pendingPhone, byId("verificationCode").value.trim());
      state.user.phone = result.phone;
      state.user.phoneVerified = true;
      showMessage("Phone number verified.", "success");
      render();
    } else if (action === "start-location-sharing") {
      if (!navigator.geolocation) throw new Error("This browser does not support geolocation.");
      const tripId = byId("navigationTripSelect").value;
      if (!tripId) throw new Error("Choose an active trip before starting location sharing.");
      if (state.locationWatchId !== null) navigator.geolocation.clearWatch(state.locationWatchId);
      let lastSentAt = 0;
      state.locationWatchId = navigator.geolocation.watchPosition(async (position) => {
        state.lastLocation = {
          latitude: position.coords.latitude,
          longitude: position.coords.longitude,
          accuracyMeters: position.coords.accuracy,
          speedMps: position.coords.speed,
          headingDegrees: position.coords.heading,
          ...(tripId ? { tripId } : {}),
        };
        byId("driverLocationStatus").textContent = `GPS fix ±${Math.round(position.coords.accuracy)} m · ${new Date(position.timestamp).toLocaleTimeString()}`;
        if (Date.now() - lastSentAt >= 10000) {
          lastSentAt = Date.now();
          try {
            await api.updateDriverLocation(state.lastLocation);
            byId("driverLocationStatus").textContent = `Sharing location${tripId ? " with the rider on this trip" : ""} · ${new Date().toLocaleTimeString()}`;
          } catch (error) {
            byId("driverLocationStatus").textContent = `Location update failed: ${error.message}`;
          }
        }
      }, (error) => {
        byId("driverLocationStatus").textContent = `Location permission/error: ${error.message}`;
        stopLocationMonitoring();
      }, { enableHighAccuracy: true, maximumAge: 5000, timeout: 15000 });
      byId("driverLocationStatus").textContent = "Waiting for GPS permission and a location fix…";
    } else if (action === "stop-location-sharing") {
      stopLocationMonitoring();
      state.lastLocation = null;
      byId("driverLocationStatus").textContent = "Location sharing is stopped.";
      showMessage("Location sharing stopped.", "success");
    } else if (action === "assign-bay") {
      const assignment = await api.assignBay(value);
      showMessage(`Pickup bay ${assignment.label} assigned at ${assignment.stationId}.`, "success");
    } else if (action === "mark-bay-arrival") {
      const assignment = await api.markBayArrival(value);
      showMessage(`Arrival recorded at pickup bay ${assignment.label}.`, "success");
    } else if (action === "download-driver-document") {
      const blob = await api.downloadDriverDocument(value);
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = button.textContent.trim();
      link.click();
      setTimeout(() => URL.revokeObjectURL(url), 0);
    } else if (action === "review-driver-document") {
      const note = button.dataset.status === "rejected" ? (prompt("Optional reason for rejection") || "") : "";
      await api.reviewDriverDocument(value, button.dataset.status, note);
      renderDriverDocuments(byId("adminDriverDocuments"), await api.adminDriverDocuments(), true);
      showMessage(`Document ${button.dataset.status}.`, "success");
    } else if (action === "toggle-bay") {
      const bay = (await api.adminBays()).find((item) => item.id === value);
      if (!bay) throw new Error("Bay no longer exists. Refresh and try again.");
      await api.saveBay({ ...bay, open: button.dataset.open === "true" });
      renderAdminBays(byId("adminBayList"), await api.adminBays());
      showMessage(`Bay ${button.dataset.open === "true" ? "opened" : "closed"} for assignment.`, "success");
    } else if (action === "toggle-subscription-plan") {
      await api.setSubscriptionPlanActive(value, button.dataset.active === "true");
      renderSubscriptionPlans(byId("adminSubscriptionPlanList"), await api.adminSubscriptionPlans(), null, true);
      showMessage("Plan availability updated.", "success");
    } else if (action === "subscribe") {
      if (!button.dataset.idempotencyKey) {
        button.dataset.idempotencyKey = `subscription-${globalThis.crypto?.randomUUID?.() || `${Date.now()}-${Math.random().toString(16).slice(2)}`}`;
      }
      await api.subscribe(value, button.dataset.idempotencyKey);
      render();
      showMessage("Subscription purchased using your wallet balance.", "success");
    } else if (action === "cancel-subscription") {
      await api.cancelSubscription(value);
      render();
      showMessage("Subscription cancelled.", "success");
    } else if (action === "support-ticket-status") {
      const status = button.closest("tr").querySelector("select").value;
      await api.updateSupportTicketStatus(value, status);
      showMessage("Support ticket status updated.", "success");
      render();
    } else if (action === "pay-trip") {
      state.selectedTripId = value;
      navigate(pathFor("rider", "payment", `tripId=${encodeURIComponent(value)}`));
    } else if (action === "confirm-payment") {
      state.lastPayment = await api.payTrip(value, `portal-trip-payment-${value}`);
      state.paidTripIds.add(value);
      state.selectedTripId = value;
      navigate(pathFor("rider", "payment-success"));
    } else if (action === "cancel-trip") {
      await api.updateTripStatus(value, "cancelled");
      showMessage("Trip request cancelled.", "success");
      render();
    } else if (action === "run-simulation") {
      const result = await api.runSimulation();
      byId("simulationOutput").textContent = JSON.stringify(result, null, 2);
      showMessage("Simulation completed.", "success");
    } else if (action === "run-dispatch") {
      const result = await api.dispatchPending();
      byId("dispatchOutput").textContent = JSON.stringify(result, null, 2);
      showMessage("Dispatch completed.", "success");
    } else if (action === "export-trips") {
      const trips = await api.listTrips();
      const columns = ["id", "riderId", "driverId", "originId", "destination", "status", "poolMode", "distanceMeters", "farePaise", "createdAt"];
      const csvCell = (value) => `"${String(value ?? "").replaceAll('"', '""')}"`;
      const csv = [columns.map(csvCell).join(","), ...trips.map((trip) => columns.map((column) => csvCell(trip[column])).join(","))].join("\r\n");
      const link = document.createElement("a");
      const url = URL.createObjectURL(new Blob([csv], { type: "text/csv;charset=utf-8" }));
      link.href = url;
      link.download = `nimma-saarthi-trips-${new Date().toISOString().slice(0, 10)}.csv`;
      link.click();
      setTimeout(() => URL.revokeObjectURL(url), 0);
      showMessage(`Exported ${trips.length} trips as CSV.`, "success");
    } else if (action === "driver-availability") {
      await api.updateDriverAvailability(value);
      showMessage(`Availability set to ${value}.`, "success");
      render();
    } else if (action === "driver-trip-status") {
      const trip = await api.updateTripStatus(value, button.dataset.status);
      state.selectedTripId = trip.id;
      showMessage(`Trip status updated to ${trip.status}.`, "success");
      if (parsePortalRoute(location.hash)?.page === "assignment") render();
      else navigate(pathFor("driver", "trip-summary", `tripId=${encodeURIComponent(trip.id)}`));
    }
  } catch (error) {
    showMessage(error instanceof ApiError ? error.message : error.message, "error");
  } finally {
    if (button.isConnected) button.disabled = false;
  }
}

async function checkHealth() {
  try {
    const result = await api.health();
    const healthy = result?.status === "ok";
    byId("healthDot").className = `status-dot ${healthy ? "status-dot-green" : "status-dot-red"}`;
    byId("healthText").textContent = healthy ? "Service online" : "Service degraded";
  } catch {
    byId("healthDot").className = "status-dot status-dot-red";
    byId("healthText").textContent = "API unavailable";
  }
}

document.addEventListener("submit", (event) => { void handleSubmit(event); });
document.addEventListener("click", (event) => { void handleClick(event); });
document.addEventListener("change", (event) => {
  if (event.target.closest("#tripForm")) {
    state.quote = null;
    state.tripInput = null;
    byId("bookingCreateTrip")?.classList.add("hidden");
    const requestButton = byId("bookingCreateTrip");
    if (requestButton) requestButton.disabled = true;
    const output = byId("bookingQuoteOutput");
    if (output) output.textContent = "Journey options changed. Recalculate the route and fare before requesting the trip.";
  }
});
byId("signOutButton").addEventListener("click", clearSession);
window.addEventListener("hashchange", render);
byId("apiOriginLabel").textContent = apiBase;
void checkHealth();
render();