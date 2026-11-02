export class ApiError extends Error {
  constructor(message, status, payload) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.payload = payload;
  }
}

export class ApiClient {
  constructor(baseUrl, fetchImpl = globalThis.fetch) {
    if (typeof baseUrl !== "string" || !baseUrl.trim()) throw new TypeError("API base URL is required");
    if (typeof fetchImpl !== "function") throw new TypeError("fetch implementation is required");
    this.baseUrl = baseUrl.replace(/\/+$/, "");
    this.fetchImpl = fetchImpl.bind(globalThis);
    this.token = null;
  }

  setToken(token) {
    this.token = token || null;
  }

  clearSession() {
    this.token = null;
  }

  async request(path, { method = "GET", body, auth = true, idempotencyKey } = {}) {
    const headers = { Accept: "application/json" };
    const formData = typeof FormData !== "undefined" && body instanceof FormData;
    if (body !== undefined && !formData) headers["Content-Type"] = "application/json";
    if (auth && this.token) headers.Authorization = `Bearer ${this.token}`;
    if (idempotencyKey) headers["Idempotency-Key"] = idempotencyKey;

    let response;
    try {
      response = await this.fetchImpl(`${this.baseUrl}${path}`, {
        method,
        headers,
        mode: "cors",
        body: body === undefined ? undefined : formData ? body : JSON.stringify(body),
      });
    } catch (error) {
      throw new ApiError(`Could not reach the API at ${this.baseUrl}. Check that the backend is running and CORS allows this page.`, 0, error);
    }

    const text = response.status === 204 ? "" : await response.text();
    let payload = null;
    if (text) {
      try {
        payload = JSON.parse(text);
      } catch {
        payload = { message: text };
      }
    }
    if (!response.ok) {
      const message = payload?.message || `Request failed (${response.status})`;
      throw new ApiError(message, response.status, payload);
    }
    return payload?.data ?? payload;
  }

  async health() {
    return this.request("/health", { auth: false });
  }

  async stations() {
    return this.request("/stations", { auth: false });
  }

  async transitRoute({ originId, destinationId, departureTime }) {
    const query = new URLSearchParams({ originId, destinationId });
    if (departureTime) query.set("departureTime", departureTime);
    return this.request(`/transit/route?${query}`);
  }

  async publicMetrics() {
    return this.request("/public/metrics", { auth: false });
  }

  async register({ name, email, phone, password, role = "rider", invitationCode }) {
    return this.request("/auth/register", { method: "POST", auth: false, body: { name, email, phone, password, role, invitationCode } });
  }

  async login({ email, password, role }) {
    return this.request("/auth/login", { method: "POST", auth: false, body: { email, password, role } });
  }

  async quoteFare({ tripId, poolMode, pickupLatitude, pickupLongitude, dropoffLatitude, dropoffLongitude }) {
    return this.request("/fare/quote", { method: "POST", body: { tripId, poolMode, pickupLatitude, pickupLongitude, dropoffLatitude, dropoffLongitude } });
  }

  async createTrip({ originId, destination, poolMode, pickupLatitude, pickupLongitude, dropoffLatitude, dropoffLongitude, partySize, requiresAccessible }) {
    return this.request("/trips", { method: "POST", body: { originId, destination, poolMode, pickupLatitude, pickupLongitude, dropoffLatitude, dropoffLongitude, partySize, requiresAccessible } });
  }

  async listTrips() {
    return this.request("/trips");
  }

  async getTrip(tripId) {
    return this.request(`/trips/${encodeURIComponent(tripId)}`);
  }

  async updateTripStatus(tripId, status) {
    return this.request(`/trips/${encodeURIComponent(tripId)}/status`, { method: "PATCH", body: { status } });
  }

  async matchPool() {
    return this.request("/pools/match", { method: "POST", body: {} });
  }

  async wallet() {
    return this.request("/wallet");
  }

  async payTrip(tripId, idempotencyKey) {
    return this.request(`/trips/${encodeURIComponent(tripId)}/pay`, { method: "POST", idempotencyKey });
  }

  async metrics() {
    return this.request("/metrics");
  }

  async runSimulation() {
    return this.request("/simulation/run", { method: "POST", body: {} });
  }

  async riders() {
    return this.request("/riders");
  }

  async drivers() {
    return this.request("/drivers");
  }

  async vehicles() {
    return this.request("/vehicles");
  }

  async dispatchPending() {
    return this.request("/admin/dispatch/run", { method: "POST", body: {} });
  }

  async verifyDriver(userId, verified) {
    return this.request(`/admin/drivers/${encodeURIComponent(userId)}/verification`, {
      method: "PATCH",
      body: { verified },
    });
  }

  async assignDriverVehicle(userId, vehicleId) {
    return this.request(`/admin/drivers/${encodeURIComponent(userId)}/vehicle`, {
      method: "PATCH",
      body: { vehicleId },
    });
  }

  async supportTickets() {
    return this.request("/support/tickets");
  }

  async createSupportTicket({ category, subject, message }) {
    return this.request("/support/tickets", { method: "POST", body: { category, subject, message } });
  }

  async adminSupportTickets() {
    return this.request("/admin/support/tickets");
  }

  async updateSupportTicketStatus(ticketId, status) {
    return this.request(`/admin/support/tickets/${encodeURIComponent(ticketId)}/status`, {
      method: "PATCH",
      body: { status },
    });
  }

  async rateTrip(tripId, { stars, comment }) {
    return this.request(`/trips/${encodeURIComponent(tripId)}/rating`, {
      method: "POST",
      body: { stars, comment },
    });
  }

  async tripRating(tripId) {
    return this.request(`/trips/${encodeURIComponent(tripId)}/rating`);
  }

  async tripLocation(tripId) {
    return this.request(`/trips/${encodeURIComponent(tripId)}/location`);
  }

  async poolRoutePreview(tripId) {
    return this.request(`/trips/${encodeURIComponent(tripId)}/pool-route`);
  }

  async navigationRoute({ startLatitude, startLongitude, endLatitude, endLongitude }) {
    return this.request("/navigation/route", {
      method: "POST",
      body: { startLatitude, startLongitude, endLatitude, endLongitude },
    });
  }

  async startPhoneVerification(phone) {
    return this.request("/auth/otp/start", { method: "POST", body: { phone } });
  }

  async checkPhoneVerification(phone, code) {
    return this.request("/auth/otp/check", { method: "POST", body: { phone, code } });
  }

  async driverProfile() {
    return this.request("/drivers/me");
  }

  async updateDriverAvailability(status) {
    return this.request("/drivers/me/availability", { method: "PATCH", body: { status } });
  }

  async updateDriverLocation(location) {
    return this.request("/drivers/me/location", { method: "POST", body: location });
  }

  async driverDocuments() {
    return this.request("/drivers/me/documents");
  }

  async uploadDriverDocument(formData) {
    return this.request("/drivers/me/documents", { method: "POST", body: formData });
  }

  async adminDriverDocuments() {
    return this.request("/admin/driver-documents");
  }

  async reviewDriverDocument(documentId, status, note = "") {
    return this.request(`/admin/driver-documents/${encodeURIComponent(documentId)}/review`, {
      method: "PATCH",
      body: { status, note },
    });
  }

  async downloadDriverDocument(documentId) {
    if (!this.token) throw new ApiError("Sign in again before downloading a driver document.", 401, null);
    const fetchWithSessionToken = (url, options) => this.fetchImpl(url, {
      ...options,
      headers: { ...options.headers, Authorization: `Bearer ${this.token}` },
    });
    let response;
    try {
      response = await fetchWithSessionToken(`${this.baseUrl}/admin/driver-documents/${encodeURIComponent(documentId)}/download`, {
        headers: { Authorization: `******`, Accept: "application/octet-stream" },
        mode: "cors",
      });
    } catch (error) {
      throw new ApiError(`Could not reach the API at ${this.baseUrl}.`, 0, error);
    }
    if (!response.ok) {
      const payload = await response.json().catch(() => null);
      throw new ApiError(payload?.message || `Request failed (${response.status})`, response.status, payload);
    }
    return response.blob();
  }

  async driverEarnings() {
    return this.request("/drivers/me/earnings");
  }

  async driverRatingSummary() {
    return this.request("/drivers/me/ratings");
  }

  async assignBay(tripId) {
    return this.request(`/drivers/me/trips/${encodeURIComponent(tripId)}/bay`, { method: "POST", body: {} });
  }

  async markBayArrival(tripId) {
    return this.request(`/drivers/me/trips/${encodeURIComponent(tripId)}/bay-arrival`, { method: "PATCH", body: {} });
  }

  async adminBays() {
    return this.request("/admin/bays");
  }

  async saveBay(bay) {
    return this.request("/admin/bays", { method: "POST", body: bay });
  }

  async subscriptionPlans() {
    return this.request("/subscriptions/plans", { auth: false });
  }

  async adminSubscriptionPlans() {
    return this.request("/admin/subscription-plans");
  }

  async saveSubscriptionPlan(plan) {
    return this.request("/admin/subscription-plans", { method: "POST", body: plan });
  }

  async setSubscriptionPlanActive(planId, active) {
    return this.request(`/admin/subscription-plans/${encodeURIComponent(planId)}/active`, {
      method: "PATCH",
      body: { active },
    });
  }

  async currentSubscription() {
    return this.request("/subscriptions/me");
  }

  async subscribe(planId, idempotencyKey) {
    return this.request("/subscriptions", { method: "POST", body: { planId }, idempotencyKey });
  }

  async cancelSubscription(subscriptionId) {
    return this.request(`/subscriptions/${encodeURIComponent(subscriptionId)}`, { method: "DELETE" });
  }

  async creditWallet(userId, { amountPaise, description }, idempotencyKey) {
    return this.request(`/admin/wallets/${encodeURIComponent(userId)}/credits`, {
      method: "POST",
      body: { amountPaise, description },
      idempotencyKey,
    });
  }

  async createInvitation(role) {
    return this.request("/admin/invitations", { method: "POST", body: { role } });
  }
}
