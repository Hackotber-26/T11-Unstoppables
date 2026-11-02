export const AUTH_PAGES = [
  { id: "welcome", label: "Welcome", title: "Choose your workspace", description: "Select a rider, driver, or operations account." },
  { id: "login-rider", label: "Rider sign in", title: "Rider sign in", description: "Sign in to plan and manage your journeys." },
  { id: "register-rider", label: "Rider registration", title: "Create a rider account", description: "Create a rider profile and wallet." },
  { id: "login-driver", label: "Driver sign in", title: "Driver sign in", description: "Sign in to your driver workspace." },
  { id: "register-driver", label: "Driver registration", title: "Create a driver account", description: "Driver registration requires an invitation and starts unverified." },
  { id: "login-admin", label: "Admin sign in", title: "Operations sign in", description: "Sign in to the protected operations workspace." },
  { id: "register-admin", label: "Admin registration", title: "Create an operations account", description: "Admin registration requires a single-use invitation from an existing administrator." },
  { id: "consent", label: "Consent", title: "Privacy and consent", description: "Review the service limits before sharing account information." },
  { id: "onboarding", label: "Onboarding", title: "Account setup", description: "Your rider profile is created during registration." },
  { id: "otp-verification", label: "Verification", title: "Phone verification", description: "Verify a phone number using the configured SMS provider from your rider or driver profile." },
];

export const PORTAL_PAGES = {
  rider: [
    { id: "dashboard", label: "Overview", title: "Rider dashboard", description: "Your trips, wallet, and next steps." },
    { id: "transit", label: "Train and metro times", title: "Plan with transit", description: "Request a Google Maps transit itinerary and scheduled train or metro departure times." },
    { id: "destination-select", label: "Plan a trip", title: "Journey details", description: "Choose a metro station and map pins for a route calculated by the backend." },
    { id: "fare-preview", label: "Fare quote", title: "Review your fare", description: "The server calculates the authoritative quote." },
    { id: "matching", label: "Find a match", title: "Corridor matching", description: "Check eligible candidates for your requested trip." },
    { id: "live-trip", label: "Live trip", title: "Trip details", description: "Inspect the latest status returned by the API." },
    { id: "payment", label: "Payment", title: "Pay from wallet", description: "Use your available wallet balance for an eligible trip." },
    { id: "payment-success", label: "Payment receipt", title: "Payment result", description: "The latest payment response from the API." },
    { id: "trip-history", label: "Trip history", title: "Your trips", description: "Review and manage trips attached to your account." },
    { id: "wallet", label: "Wallet", title: "Wallet and ledger", description: "View the current balance and recorded entries." },
    { id: "profile", label: "Profile", title: "Account profile", description: "Account details returned at sign-in." },
    { id: "rating", label: "Rating", title: "Trip feedback", description: "Rate completed trips and share feedback with the driver." },
    { id: "support", label: "Support", title: "Support", description: "Create a support request and follow its status." },
    { id: "subscriptions", label: "Subscriptions", title: "Ride subscriptions", description: "Purchase and manage ride discounts using your wallet balance." },
  ],
  driver: [
    { id: "dashboard", label: "Overview", title: "Driver dashboard", description: "Driver workspace and dispatch connection status." },
    { id: "availability", label: "Availability", title: "Availability", description: "Set your availability after operations verifies your account and assigns a vehicle." },
    { id: "assignment", label: "Assignments", title: "Trip assignments", description: "Review trips assigned to your driver account and update their status." },
    { id: "navigation", label: "Navigation", title: "Route navigation", description: "Share your location on an active trip and request a road route from the local OSRM service." },
    { id: "bay-arrival", label: "Bay arrival", title: "Station bay arrival", description: "Reserve the best available station pickup bay and record arrival." },
    { id: "otp-check", label: "Pickup check", title: "Pickup verification", description: "Rider pickup verification is not connected yet." },
    { id: "trip-summary", label: "Trip summary", title: "Trip summary", description: "Review trip details and start or complete an assigned trip." },
    { id: "earnings", label: "Earnings", title: "Earnings", description: "Review completed-trip gross fares, estimated commission, and estimated driver share." },
    { id: "ratings", label: "Ratings", title: "Driver ratings", description: "Review your average rider rating and feedback count." },
    { id: "profile", label: "Profile", title: "Driver profile", description: "Driver profile editing is not available from the current API." },
    { id: "verification", label: "Verification", title: "Driver verification", description: "Upload private license, identity, registration, and insurance documents for administrator review." },
  ],
  admin: [
    { id: "dashboard", label: "Overview", title: "Operations dashboard", description: "Live aggregates and links to operational views." },
    { id: "analytics", label: "Analytics", title: "Analytics", description: "Backend trip metrics and deterministic simulation." },
    { id: "trips", label: "Trips", title: "Trip operations", description: "Review trips returned by the admin API." },
    { id: "pools", label: "Pools", title: "Pool operations", description: "Form corridor pools and dispatch eligible solo trips." },
    { id: "drivers", label: "Drivers", title: "Driver directory", description: "Driver directory from the admin API." },
    { id: "users", label: "Users", title: "User directory", description: "Rider and driver directories from the admin API." },
    { id: "stations", label: "Stations", title: "Stations", description: "Station management is not exposed by the current API." },
    { id: "bays", label: "Bays", title: "Bays", description: "Manage station pickup bays and their capacity, access, walking distance, and operating status." },
    { id: "vehicles", label: "Vehicles", title: "Vehicles", description: "Inspect active, verified, and assigned fleet vehicles." },
    { id: "incidents", label: "Support and incidents", title: "Support and incidents", description: "Review rider support tickets and update their status." },
    { id: "subscriptions", label: "Subscriptions", title: "Subscriptions", description: "Manage wallet-priced ride plans, monthly limits, discounts, and availability." },
    { id: "reports", label: "Reports", title: "Reports", description: "Download the current operations trip list as a CSV report." },
  ],
};

export function parsePortalRoute(hash) {
  const raw = String(hash || "").replace(/^#\/?/, "");
  const [path, query = ""] = raw.split("?", 2);
  const [area, page] = path.split("/");
  if (area === "auth") {
    const definition = AUTH_PAGES.find((item) => item.id === page);
    return definition ? { area, page, definition, params: new URLSearchParams(query) } : null;
  }
  if (!Object.hasOwn(PORTAL_PAGES, area)) return null;
  const definition = PORTAL_PAGES[area].find((item) => item.id === page);
  return definition ? { area, page, definition, params: new URLSearchParams(query) } : null;
}

export function pathFor(area, page, query = "") {
  return `#/${area}/${page}${query ? `?${query}` : ""}`;
}

export function homeForRole(role) {
  return Object.hasOwn(PORTAL_PAGES, role) ? pathFor(role, "dashboard") : pathFor("auth", "welcome");
}