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
    if (body !== undefined) headers["Content-Type"] = "application/json";
    if (auth && this.token) headers.Authorization = `Bearer ${this.token}`;
    if (idempotencyKey) headers["Idempotency-Key"] = idempotencyKey;

    let response;
    try {
      response = await this.fetchImpl(`${this.baseUrl}${path}`, {
        method,
        headers,
        mode: "cors",
        body: body === undefined ? undefined : JSON.stringify(body),
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

  async register({ name, email, phone, password }) {
    return this.request("/auth/register", { method: "POST", auth: false, body: { name, email, phone, password } });
  }

  async login({ email, password }) {
    return this.request("/auth/login", { method: "POST", auth: false, body: { email, password } });
  }

  async quoteFare({ tripId, poolMode, distanceMeters }) {
    return this.request("/fare/quote", { method: "POST", body: { tripId, poolMode, distanceMeters } });
  }

  async createTrip({ originId, destination, poolMode, distanceMeters }) {
    return this.request("/trips", { method: "POST", body: { originId, destination, poolMode, distanceMeters } });
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

  async creditWallet(userId, { amountCents, description }, idempotencyKey) {
    return this.request(`/admin/wallets/${encodeURIComponent(userId)}/credits`, {
      method: "POST",
      body: { amountCents, description },
      idempotencyKey,
    });
  }
}
