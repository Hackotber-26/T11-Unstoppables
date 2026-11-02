const maps = new Map();

export function normalizeRouteGeometry(geometry) {
  if (geometry?.type !== "LineString" || !Array.isArray(geometry.coordinates) || geometry.coordinates.length < 2 || geometry.coordinates.length > 20000) {
    throw new TypeError("The routing service returned invalid map geometry.");
  }
  return geometry.coordinates.map((coordinate) => {
    if (!Array.isArray(coordinate) || coordinate.length < 2) {
      throw new TypeError("The routing service returned invalid map coordinates.");
    }
    const [longitude, latitude] = coordinate;
    if (!Number.isFinite(latitude) || latitude < -90 || latitude > 90 || !Number.isFinite(longitude) || longitude < -180 || longitude > 180) {
      throw new TypeError("The routing service returned out-of-range map coordinates.");
    }
    return [latitude, longitude];
  });
}

function mountBaseMap(element, center, zoom = 14) {
  if (!globalThis.L) {
    element.textContent = "The map library is unavailable. Try reloading this page to load the map.";
    return null;
  }
  maps.get(element)?.stop();
  maps.get(element)?.remove();
  const map = globalThis.L.map(element, { scrollWheelZoom: false }).setView(center, zoom);
  globalThis.L.tileLayer("https://tile.openstreetmap.org/{z}/{x}/{y}.png", {
    maxZoom: 19,
    attribution: '&copy; <a href="https://www.openstreetmap.org/copyright" target="_blank" rel="noopener noreferrer">OpenStreetMap contributors</a>',
  }).addTo(map);
  maps.set(element, map);
  return map;
}

function scheduleMapResize(element, map) {
  requestAnimationFrame(() => {
    if (maps.get(element) === map) map.invalidateSize();
  });
}

export function showRouteMap(element, geometry) {
  const coordinates = normalizeRouteGeometry(geometry);
  const map = mountBaseMap(element, coordinates[0]);
  if (!map) return;
  globalThis.L.polyline(coordinates, { color: "#a855f7", weight: 6, opacity: 0.9 }).addTo(map);
  globalThis.L.circleMarker(coordinates[0], { radius: 7, color: "#10b981", fillOpacity: 1 }).addTo(map).bindTooltip("Start");
  globalThis.L.circleMarker(coordinates.at(-1), { radius: 7, color: "#f43f5e", fillOpacity: 1 }).addTo(map).bindTooltip("Destination");
  map.fitBounds(globalThis.L.latLngBounds(coordinates), { padding: [24, 24], animate: false });
  scheduleMapResize(element, map);
}

export function showPoolRouteMap(element, geometry, stops) {
  const coordinates = normalizeRouteGeometry(geometry);
  const map = mountBaseMap(element, coordinates[0]);
  if (!map) return;
  const layers = [globalThis.L.polyline(coordinates, { color: "#a855f7", weight: 6, opacity: 0.9 })];
  for (const stop of stops) {
    if (!Number.isFinite(stop.latitude) || stop.latitude < -90 || stop.latitude > 90 ||
        !Number.isFinite(stop.longitude) || stop.longitude < -180 || stop.longitude > 180) {
      throw new TypeError("The pool route returned an invalid stop location.");
    }
    layers.push(globalThis.L.circleMarker([stop.latitude, stop.longitude], {
      radius: stop.isYourStop ? 9 : 7,
      color: stop.isYourStop ? "#10b981" : "#f59e0b",
      fillOpacity: 1,
    }).bindTooltip(stop.isYourStop ? "Your drop-off" : `Anonymous passenger drop-off ${stop.sequence}`));
  }
  const group = globalThis.L.featureGroup(layers).addTo(map);
  map.fitBounds(group.getBounds(), { padding: [24, 24], animate: false });
  scheduleMapResize(element, map);
}

export function showLocationMap(element, latitude, longitude) {
  if (!Number.isFinite(latitude) || latitude < -90 || latitude > 90 || !Number.isFinite(longitude) || longitude < -180 || longitude > 180) {
    throw new TypeError("The location service returned out-of-range map coordinates.");
  }
  const point = [latitude, longitude];
  const map = mountBaseMap(element, point, 16);
  if (!map) return;
  globalThis.L.circleMarker(point, { radius: 9, color: "#10b981", fillColor: "#10b981", fillOpacity: 0.85 }).addTo(map).bindTooltip("Driver location");
  scheduleMapResize(element, map);
}

export function createBookingMap(element, onPickup, onDropoff) {
  const map = mountBaseMap(element, [12.9716, 77.5946], 12);
  if (!map) return null;
  const markers = { pickup: null, dropoff: null };
  let routeLayer = null;
  let selection = "pickup";

  function setPoint(type, point) {
    const marker = markers[type];
    if (marker) map.removeLayer(marker);
    markers[type] = point
      ? globalThis.L.marker([point.latitude, point.longitude], { keyboard: true }).addTo(map).bindTooltip(type === "pickup" ? "Your pickup" : "Your drop-off")
      : null;
    if (type === "pickup") onPickup(point);
    else onDropoff(point);
    if (routeLayer) {
      map.removeLayer(routeLayer);
      routeLayer = null;
    }
  }

  map.on("click", (event) => {
    const point = { latitude: event.latlng.lat, longitude: event.latlng.lng };
    setPoint(selection, point);
  });
  scheduleMapResize(element, map);
  return {
    selectPoint(type) {
      if (type !== "pickup" && type !== "dropoff") throw new TypeError("Map selection must be pickup or drop-off.");
      selection = type;
    },
    setPoint,
    focusPoint(point) {
      if (!Number.isFinite(point.latitude) || !Number.isFinite(point.longitude)) throw new TypeError("Map focus coordinates are invalid.");
      map.setView([point.latitude, point.longitude], 15);
    },
    resize() {
      scheduleMapResize(element, map);
    },
    setRoute(geometry) {
      const coordinates = normalizeRouteGeometry(geometry);
      if (routeLayer) map.removeLayer(routeLayer);
      routeLayer = globalThis.L.polyline(coordinates, { color: "#a855f7", weight: 6, opacity: 0.9 }).addTo(map);
      map.fitBounds(routeLayer.getBounds(), { padding: [24, 24], animate: false });
      scheduleMapResize(element, map);
    },
    setPoolRoute(geometry, stops) {
      const coordinates = normalizeRouteGeometry(geometry);
      if (routeLayer) map.removeLayer(routeLayer);
      const layers = [globalThis.L.polyline(coordinates, { color: "#a855f7", weight: 6, opacity: 0.9 })];
      for (const stop of stops) {
        if (!Number.isFinite(stop.latitude) || !Number.isFinite(stop.longitude) ||
            stop.latitude < -90 || stop.latitude > 90 || stop.longitude < -180 || stop.longitude > 180) {
          throw new TypeError("The pool route returned an invalid stop location.");
        }
        const label = stop.isYourStop ? "Your drop-off" : `Passenger drop-off ${stop.sequence}`;
        layers.push(globalThis.L.circleMarker([stop.latitude, stop.longitude], {
          radius: stop.isYourStop ? 9 : 7,
          color: stop.isYourStop ? "#10b981" : "#f59e0b",
          fillOpacity: 1,
        }).bindTooltip(label));
      }
      routeLayer = globalThis.L.featureGroup(layers).addTo(map);
      map.fitBounds(routeLayer.getBounds(), { padding: [24, 24], animate: false });
      scheduleMapResize(element, map);
    },
  };
}

export function clearRouteMaps() {
  for (const element of maps.keys()) clearRouteMap(element);
}

export function clearRouteMap(element) {
  const map = maps.get(element);
  if (!map) return;
  map.stop();
  map.remove();
  maps.delete(element);
}
