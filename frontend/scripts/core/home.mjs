import { ApiClient, ApiError } from "./api.mjs?v=2";
import { clearRouteMap, createBookingMap, showPoolRouteMap } from "./route-map.mjs";

const apiOverride = new URLSearchParams(location.search).get("api");
const requestedView = new URLSearchParams(location.search).get("view");
const apiBase = window.NIMMA_API_BASE || apiOverride || `${location.protocol}//${location.hostname || "localhost"}:8080/api/v1`;
const api = new ApiClient(apiBase);
const state = { user: null, quote: null, quoteKey: "", trips: [], paidTripIds: new Set(), messageTimer: null, bookingMap: null, bookingPoints: { pickup: null, dropoff: null }, mapSelection: "pickup" };
const byId = (id) => document.getElementById(id);
const money = (paise, currency = "INR") => new Intl.NumberFormat("en-IN", { style: "currency", currency }).format((Number(paise) || 0) / 100);

function showMessage(message, type = "") {
  const region = byId("appMessage");
  region.textContent = message;
  region.className = `app-message is-visible${type ? ` is-${type}` : ""}`;
  clearTimeout(state.messageTimer);
  state.messageTimer = setTimeout(() => { region.className = "app-message"; }, 6500);
}

function showAuthMessage(message, type = "") {
  const region = byId("authMessage");
  region.textContent = message;
  region.className = `form-message${type ? ` is-${type}` : ""}`;
}

function setView(name) {
  document.querySelectorAll("[data-view]").forEach((view) => view.classList.toggle("hidden", view.dataset.view !== name));
  document.querySelectorAll("[data-view-target]").forEach((button) => {
    const active = button.dataset.viewTarget === name;
    button.classList.toggle("is-current", active);
    if (active) button.setAttribute("aria-current", "page");
    else button.removeAttribute("aria-current");
  });
  if (name === "book") requestAnimationFrame(() => state.bookingMap?.resize());
}

function setAuthTab(tab) {
  const register = tab === "register";
  byId("loginForm").classList.toggle("hidden", register);
  byId("registerForm").classList.toggle("hidden", !register);
  document.querySelectorAll("[data-auth-tab]").forEach((button) => {
    const active = button.dataset.authTab === tab;
    button.classList.toggle("is-active", active);
    button.setAttribute("aria-selected", String(active));
  });
  showAuthMessage("");
}

function idempotencyKey(prefix) {
  const random = globalThis.crypto?.randomUUID?.() || `${Date.now()}-${Math.random().toString(16).slice(2)}`;
  return `${prefix}-${random}`;
}

function clearSession() {
  api.clearSession();
  state.user = null;
  state.quote = null;
  state.trips = [];
  state.paidTripIds = new Set();
  byId("workspaceView").classList.add("hidden");
  byId("authView").classList.remove("hidden");
  byId("signOutButton").classList.add("hidden");
  byId("adminNavButton").classList.add("hidden");
  byId("identityLabel").textContent = "Signed out";
}

function applySession(session) {
  if (!session?.accessToken || !session?.user?.id) throw new Error("The API returned an incomplete session.");
  api.setToken(session.accessToken);
  state.user = session.user;
  byId("authView").classList.add("hidden");
  byId("workspaceView").classList.remove("hidden");
  byId("signOutButton").classList.remove("hidden");
  byId("identityLabel").textContent = `${session.user.name} · ${session.user.role}`;
  byId("adminNavButton").classList.toggle("hidden", session.user.role !== "admin");
  byId("apiOriginLabel").textContent = apiBase;
  const openAdmin = requestedView === "admin" && session.user.role === "admin";
  setView(openAdmin ? "admin" : "book");
  refreshWorkspace();
  if (openAdmin) loadAdmin().catch(handleError);
  return openAdmin;
}

function escapeDisplay(value) {
  return String(value ?? "");
}

function tripCard(trip) {
  const article = document.createElement("article");
  article.className = "trip-row";
  const route = document.createElement("div");
  route.className = "trip-route";
  const title = document.createElement("strong");
  title.textContent = `${escapeDisplay(trip.originId)} → ${escapeDisplay(trip.destination)}`;
  const meta = document.createElement("span");
  meta.className = "trip-meta";
  const created = trip.createdAt ? new Date(trip.createdAt).toLocaleString() : "Just requested";
  meta.textContent = `${created} · ${Number(trip.distanceMeters).toLocaleString()} m · ${trip.poolMode} · ${money(trip.farePaise)}`;
  const status = document.createElement("span");
  status.className = "status-pill";
  status.textContent = trip.status;
  route.append(title, meta, status);

  const actions = document.createElement("div");
  actions.className = "trip-actions";
  actions.append(actionButton("Refresh details", "details", trip.id, "button button-secondary button-small"));
  if (trip.status === "requested" && state.user?.role === "rider") {
    actions.append(actionButton("Find candidates", "match", trip.id, "button button-secondary button-small"));
    actions.append(actionButton("Cancel", "cancel", trip.id, "button button-quiet button-small"));
  }
  if (trip.farePaise > 0 && trip.status !== "cancelled" && !state.paidTripIds.has(trip.id) && state.user?.role === "rider") {
    actions.append(actionButton("Pay from wallet", "pay", trip.id, "button button-primary button-small"));
  }
  if (state.paidTripIds.has(trip.id)) {
    const paid = document.createElement("span");
    paid.className = "status-pill";
    paid.textContent = "paid from wallet";
    actions.append(paid);
  }
  const detail = document.createElement("p");
  detail.className = "trip-detail hidden";
  detail.dataset.tripDetail = trip.id;
  article.append(route, actions, detail);
  return article;
}

function actionButton(label, action, tripId, className) {
  const button = document.createElement("button");
  button.type = "button";
  button.className = className;
  button.textContent = label;
  button.dataset.tripAction = action;
  button.dataset.tripId = tripId;
  return button;
}

function renderTrips(trips) {
  state.trips = Array.isArray(trips) ? trips : [];
  const list = byId("tripList");
  list.replaceChildren();
  if (!state.trips.length) {
    const empty = document.createElement("div");
    empty.className = "panel empty-state";
    empty.textContent = "No trips yet. Create a request to begin.";
    list.append(empty);
    return;
  }
  state.trips.forEach((trip) => list.append(tripCard(trip)));
}

async function loadTrips() {
  document.querySelectorAll(".pool-route-map").forEach(clearRouteMap);
  const trips = await api.listTrips();
  renderTrips(trips);
}

async function loadWallet() {
  const wallet = await api.wallet();
  byId("walletBalance").textContent = money(wallet.balancePaise, wallet.currency || "INR");
  byId("walletCurrency").textContent = wallet.currency || "INR";
  const entries = Array.isArray(wallet.entries) ? wallet.entries : [];
  state.paidTripIds = new Set(entries.filter((entry) => entry.entryType === "debit" && entry.tripId).map((entry) => entry.tripId));
  const list = byId("ledgerList");
  list.replaceChildren();
  if (!entries.length) {
    const empty = document.createElement("p");
    empty.className = "empty-state";
    empty.textContent = "No ledger entries yet.";
    list.append(empty);
  } else {
    entries.forEach((entry) => {
      const row = document.createElement("div");
      row.className = "ledger-row";
      const description = document.createElement("div");
      const title = document.createElement("strong");
      title.textContent = entry.description;
      const detail = document.createElement("small");
      detail.textContent = `${entry.entryType} · ${new Date(entry.createdAt).toLocaleString()}${entry.tripId ? ` · trip ${entry.tripId}` : ""}`;
      description.append(title, detail);
      const amount = document.createElement("span");
      amount.className = `ledger-amount ${entry.amountPaise >= 0 ? "positive" : "negative"}`;
      amount.textContent = money(entry.amountPaise, wallet.currency || "INR");
      row.append(description, amount);
      list.append(row);
    });
  }
  if (state.trips.length) renderTrips(state.trips);
}

async function refreshWorkspace() {
  if (!state.user) return;
  const results = await Promise.allSettled([loadTrips(), loadWallet()]);
  const failed = results.find((result) => result.status === "rejected");
  if (failed) handleError(failed.reason);
}

async function checkHealth() {
  const dot = byId("healthDot");
  const label = byId("healthText");
  try {
    const health = await api.health();
    const healthy = health?.status === "ok";
    dot.className = `status-dot ${healthy ? "status-dot-green" : "status-dot-red"}`;
    label.textContent = healthy ? "Service online" : "Service degraded";
    label.title = Object.entries(health?.services || {}).map(([name, value]) => `${name}: ${value}`).join(" · ");
  } catch (error) {
    dot.className = "status-dot status-dot-red";
    label.textContent = "API unavailable";
    label.title = error.message;
  }
}

function handleError(error) {
  showMessage(error instanceof ApiError ? error.message : (error?.message || "Something went wrong."), "error");
  if (error instanceof ApiError && error.status === 401 && state.user) {
    clearSession();
    showAuthMessage("Your session expired. Sign in again.", "error");
  }
}

async function loadStationCatalog() {
  const select = byId("originStationSelect");
  try {
    const stations = await api.stations();
    select.replaceChildren(new Option("Choose a station", ""));
    stations.forEach((station) => {
      const lineNames = (station.lines || []).map((line) => `${line.charAt(0).toUpperCase()}${line.slice(1)} Line`);
      select.add(new Option(`${station.name} · ${lineNames.join(" / ")}`, station.id));
    });
    if (!stations.length) select.replaceChildren(new Option("No active stations available", ""));
  } catch (error) {
    select.replaceChildren(new Option("Station catalog unavailable", ""));
    showAuthMessage(error.message, "error");
  }
}

function currentTripInput() {
  const form = byId("tripForm");
  const data = new FormData(form);
  const { pickup, dropoff } = state.bookingPoints;
  if (!pickup || !dropoff) throw new Error("Choose both pickup and drop-off pins on the map.");
  return {
    originId: String(data.get("originId") || "").trim(),
    destination: `${dropoff.latitude.toFixed(5)}, ${dropoff.longitude.toFixed(5)}`,
    poolMode: String(data.get("poolMode") || "corridor"),
    pickupLatitude: pickup.latitude,
    pickupLongitude: pickup.longitude,
    dropoffLatitude: dropoff.latitude,
    dropoffLongitude: dropoff.longitude,
  };
}

function inputKey(input) {
  return JSON.stringify(input);
}

function displayQuote(quote) {
  const currency = quote.currency || "INR";
  byId("quoteEmpty").classList.add("hidden");
  byId("quoteResult").classList.remove("hidden");
  byId("quoteTotal").textContent = money(quote.totalPaise, currency);
  byId("quoteBase").textContent = money(quote.basePaise, currency);
  byId("quoteDistance").textContent = money(quote.distancePaise, currency);
  byId("quoteDiscount").textContent = `−${money(quote.discountPaise, currency)}`;
  byId("quoteCurrency").textContent = currency;
  byId("quoteRoute").textContent = `${(quote.distanceMeters / 1000).toFixed(2)} km · ${Math.ceil(quote.durationSeconds / 60)} min estimated`;
  state.bookingMap?.setRoute(quote.geometry);
}

function updateBookingPoint(type, point) {
  state.bookingPoints[type] = point;
  state.quote = null;
  state.quoteKey = "";
  byId(type === "pickup" ? "homePickupSummary" : "homeDropoffSummary").textContent = point
    ? `${type === "pickup" ? "Pickup" : "Drop-off"}: ${point.latitude.toFixed(5)}, ${point.longitude.toFixed(5)}`
    : `${type === "pickup" ? "Pickup" : "Drop-off"}: not selected`;
  byId("createTripButton").disabled = true;
  byId("quoteResult").classList.add("hidden");
  byId("quoteEmpty").classList.remove("hidden");
  byId("quoteEmpty").textContent = "Choose both map pins and calculate a fresh route and fare.";
}

async function getQuote() {
  const form = byId("tripForm");
  if (!form.reportValidity()) return;
  let input;
  try {
    input = currentTripInput();
  } catch (error) {
    showMessage(error.message, "error");
    return;
  }
  byId("quoteButton").disabled = true;
  try {
    state.quote = await api.quoteFare(input);
    state.quoteKey = inputKey(input);
    displayQuote(state.quote);
    byId("createTripButton").disabled = false;
    showMessage("Road distance, duration and fare calculated by the backend.", "success");
  } catch (error) {
    state.quote = null;
    byId("createTripButton").disabled = true;
    handleError(error);
  } finally {
    byId("quoteButton").disabled = false;
  }
}

async function createTrip(event) {
  event.preventDefault();
  let input;
  try {
    input = currentTripInput();
  } catch (error) {
    showMessage(error.message, "error");
    return;
  }
  if (!state.quote || state.quoteKey !== inputKey(input)) {
    showMessage("Get a fresh fare quote before requesting this trip.", "error");
    return;
  }
  const button = byId("createTripButton");
  button.disabled = true;
  try {
    const trip = await api.createTrip(input);
    showMessage(`Trip ${trip.id} created. It is not yet matched or assigned.`, "success");
    state.quote = null;
    state.quoteKey = "";
    byId("tripForm").reset();
    byId("quoteResult").classList.add("hidden");
    byId("quoteEmpty").classList.remove("hidden");
    setView("trips");
    await loadTrips();
  } catch (error) {
    handleError(error);
  } finally {
    button.disabled = true;
  }
}

function renderCandidates(candidates) {
  const panel = byId("candidatePanel");
  const list = byId("candidateList");
  panel.classList.remove("hidden");
  list.replaceChildren();
  if (!Array.isArray(candidates) || candidates.length === 0) {
    const empty = document.createElement("p");
    empty.className = "empty-state";
    empty.textContent = "No eligible requests in the current matching window.";
    list.append(empty);
    return;
  }
  candidates.forEach((candidate) => {
    const row = document.createElement("div");
    row.className = "candidate-row";
    const identity = document.createElement("code");
    identity.textContent = candidate.riderId;
    const reason = document.createElement("span");
    reason.textContent = candidate.reason || "Eligible candidate";
    const score = document.createElement("strong");
    score.textContent = Number(candidate.score).toFixed(3);
    row.append(identity, reason, score);
    list.append(row);
  });
}

async function tripAction(event) {
  const button = event.target.closest("[data-trip-action]");
  if (!button) return;
  const { tripAction: action, tripId } = button.dataset;
  button.disabled = true;
  try {
    if (action === "match") {
      renderCandidates(await api.matchPool());
      byId("candidatePanel").scrollIntoView({ behavior: "smooth", block: "nearest" });
    } else if (action === "details") {
      const trip = await api.getTrip(tripId);
      const detail = button.closest(".trip-row").querySelector("[data-trip-detail]");
      clearRouteMap(detail.querySelector(".pool-route-map"));
      detail.replaceChildren();
      const summary = document.createElement("p");
      summary.textContent = `Server status: ${trip.status} · Pool: ${trip.poolMode} · Road distance: ${(trip.distanceMeters / 1000).toFixed(2)} km · Fare: ${money(trip.farePaise)} · Updated: ${trip.updatedAt ? new Date(trip.updatedAt).toLocaleString() : "not provided"}`;
      detail.append(summary);
      if (trip.poolId && ["matched", "in_progress"].includes(trip.status)) {
        const preview = await api.poolRoutePreview(tripId);
        const title = document.createElement("strong");
        title.textContent = `Shared route: +${(preview.distanceDeviationMeters / 1000).toFixed(2)} km and +${Math.ceil(preview.timeWastedSeconds / 60)} min to your stop`;
        const map = document.createElement("div");
        map.className = "route-map pool-route-map";
        map.setAttribute("role", "region");
        map.setAttribute("aria-label", "Shared route with anonymous passenger drop-off markers");
        const privacy = document.createElement("p");
        privacy.className = "muted";
        privacy.textContent = "Drop-off stops are shown without passenger names or account details.";
        detail.append(title, map, privacy);
        showPoolRouteMap(map, preview.geometry, preview.stops || []);
      }
      detail.classList.remove("hidden");
    } else if (action === "cancel") {
      await api.updateTripStatus(tripId, "cancelled");
      showMessage("Trip request cancelled.", "success");
      await loadTrips();
    } else if (action === "pay") {
      const result = await api.payTrip(tripId, `trip-payment-${tripId}`);
      showMessage(`Paid ${money(result.amountPaise)} from wallet.`, "success");
      await Promise.all([loadWallet(), loadTrips()]);
    }
  } catch (error) {
    handleError(error);
  } finally {
    button.disabled = false;
  }
}

async function loadDirectory() {
  const [riders, drivers] = await Promise.all([api.riders(), api.drivers()]);
  renderCompactList(byId("adminRiders"), riders, "No riders returned.");
  renderCompactList(byId("adminDrivers"), drivers, "No drivers returned.");
}

function renderCompactList(container, records, emptyMessage) {
  container.replaceChildren();
  if (!Array.isArray(records) || !records.length) {
    container.textContent = emptyMessage;
    return;
  }
  records.forEach((record) => {
    const line = document.createElement("div");
    line.textContent = `${record.name || record.id} · ${record.id} · ${record.status || record.role || ""}`;
    container.append(line);
  });
}

async function loadAdmin() {
  const metrics = await api.metrics();
  byId("adminMetrics").textContent = JSON.stringify(metrics, null, 2);
  await loadDirectory();
}

async function runSimulation() {
  byId("runSimulationButton").disabled = true;
  try {
    const result = await api.runSimulation();
    byId("simulationResult").textContent = JSON.stringify(result, null, 2);
  } catch (error) {
    handleError(error);
  } finally {
    byId("runSimulationButton").disabled = false;
  }
}

async function login(event) {
  event.preventDefault();
  const data = new FormData(event.currentTarget);
  const submit = event.currentTarget.querySelector("[type=submit]");
  submit.disabled = true;
  try {
    const session = await api.login({ email: data.get("email"), password: data.get("password") });
    const openAdmin = applySession(session);
    const deniedAdminView = requestedView === "admin" && !openAdmin;
    showMessage(deniedAdminView ? "Operations requires an administrator account." : `Signed in as ${session.user.name}.`, deniedAdminView ? "error" : "success");
  } catch (error) {
    showAuthMessage(error.message, "error");
  } finally {
    submit.disabled = false;
  }
}

async function register(event) {
  event.preventDefault();
  const data = new FormData(event.currentTarget);
  const submit = event.currentTarget.querySelector("[type=submit]");
  submit.disabled = true;
  try {
    const session = await api.register({
      name: data.get("name"), email: data.get("email"), phone: data.get("phone"), password: data.get("password"),
    });
    const openAdmin = applySession(session);
    const deniedAdminView = requestedView === "admin" && !openAdmin;
    showMessage(deniedAdminView ? "Operations requires an administrator account." : "Account created and signed in.", deniedAdminView ? "error" : "success");
  } catch (error) {
    showAuthMessage(error.message, "error");
  } finally {
    submit.disabled = false;
  }
}

async function creditWallet(event) {
  event.preventDefault();
  const form = event.currentTarget;
  const data = new FormData(form);
  const button = form.querySelector("[type=submit]");
  button.disabled = true;
  if (!form.dataset.idempotencyKey) form.dataset.idempotencyKey = idempotencyKey("admin-credit");
  try {
    const result = await api.creditWallet(data.get("userId"), {
      amountPaise: Math.round(Number(data.get("amountRupees")) * 100), description: data.get("description"),
    }, form.dataset.idempotencyKey);
    showMessage(`Wallet credited. New balance: ${money(result.balancePaise)}.`, "success");
    form.reset();
    delete form.dataset.idempotencyKey;
  } catch (error) {
    handleError(error);
  } finally {
    button.disabled = false;
  }
}

function bindEvents() {
  document.querySelectorAll("[data-auth-tab]").forEach((button) => button.addEventListener("click", () => setAuthTab(button.dataset.authTab)));
  document.querySelectorAll("[data-view-target]").forEach((button) => button.addEventListener("click", async () => {
    setView(button.dataset.viewTarget);
    if (button.dataset.viewTarget === "trips") await loadTrips().catch(handleError);
    if (button.dataset.viewTarget === "wallet") await loadWallet().catch(handleError);
    if (button.dataset.viewTarget === "admin") await loadAdmin().catch(handleError);
  }));
  byId("loginForm").addEventListener("submit", login);
  byId("registerForm").addEventListener("submit", register);
  byId("tripForm").addEventListener("submit", createTrip);
  byId("tripForm").addEventListener("input", () => {
    if (!state.quote) return;
    state.quote = null;
    state.quoteKey = "";
    byId("createTripButton").disabled = true;
    byId("quoteResult").classList.add("hidden");
    byId("quoteEmpty").classList.remove("hidden");
    byId("quoteEmpty").textContent = "Journey details changed. Request a fresh route and quote before creating the trip.";
  });
  document.querySelectorAll("[data-map-selection]").forEach((button) => button.addEventListener("click", () => {
    state.mapSelection = button.dataset.mapSelection;
    state.bookingMap?.selectPoint(state.mapSelection);
    document.querySelectorAll("[data-map-selection]").forEach((item) => {
      const selected = item.dataset.mapSelection === state.mapSelection;
      item.classList.toggle("is-current", selected);
      item.setAttribute("aria-pressed", String(selected));
    });
  }));
  byId("useCurrentLocationButton").addEventListener("click", async () => {
    if (!navigator.geolocation) {
      showMessage("This browser does not support device location.", "error");
      return;
    }
    try {
      const position = await new Promise((resolve, reject) => {
        navigator.geolocation.getCurrentPosition(resolve, reject, { enableHighAccuracy: true, timeout: 15000, maximumAge: 30000 });
      });
      const point = { latitude: position.coords.latitude, longitude: position.coords.longitude };
      state.bookingMap?.setPoint("pickup", point);
      state.bookingMap?.focusPoint(point);
    } catch (error) {
      showMessage(error.message || "Device location is unavailable.", "error");
    }
  });
  byId("quoteButton").addEventListener("click", getQuote);
  byId("tripList").addEventListener("click", tripAction);
  byId("refreshTripsButton").addEventListener("click", () => loadTrips().catch(handleError));
  byId("refreshWalletButton").addEventListener("click", () => loadWallet().catch(handleError));
  byId("refreshAllButton").addEventListener("click", () => refreshWorkspace());
  byId("refreshAdminButton").addEventListener("click", () => loadAdmin().catch(handleError));
  byId("loadDirectoryButton").addEventListener("click", () => loadDirectory().catch(handleError));
  byId("runSimulationButton").addEventListener("click", runSimulation);
  byId("creditForm").addEventListener("submit", creditWallet);
  byId("closeCandidatesButton").addEventListener("click", () => byId("candidatePanel").classList.add("hidden"));
  byId("signOutButton").addEventListener("click", () => {
    clearSession();
    showAuthMessage("Signed out. The session token was cleared.", "success");
  });
}

state.bookingMap = createBookingMap(byId("homeBookingMap"),
  (point) => updateBookingPoint("pickup", point),
  (point) => updateBookingPoint("dropoff", point));
bindEvents();
byId("apiOriginLabel").textContent = apiBase;
checkHealth();
loadStationCatalog();
setInterval(checkHealth, 20000);
