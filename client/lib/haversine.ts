import type { LatLng } from "./types";

/**
 * Great-circle distance between two points using the Haversine formula.
 *
 * The Earth is treated as a sphere of radius 6371 km. Accuracy is more than
 * good enough for a guessing game (sub-0.5% error vs. the WGS84 ellipsoid).
 *
 * @returns distance in kilometers, rounded to 1 decimal place.
 */
export function haversineDistanceKm(a: LatLng, b: LatLng): number {
  const EARTH_RADIUS_KM = 6371;

  // Convert degrees → radians.
  const toRad = (deg: number): number => (deg * Math.PI) / 180;

  const lat1 = toRad(a.lat);
  const lat2 = toRad(b.lat);
  const dLat = toRad(b.lat - a.lat);
  const dLng = toRad(b.lng - a.lng);

  // Haversine core.
  const h =
    Math.sin(dLat / 2) ** 2 +
    Math.cos(lat1) * Math.cos(lat2) * Math.sin(dLng / 2) ** 2;

  // atan2-based central angle; multiply by radius for arc length.
  const c = 2 * Math.atan2(Math.sqrt(h), Math.sqrt(1 - h));
  const distanceKm = EARTH_RADIUS_KM * c;

  return Math.round(distanceKm * 10) / 10;
}
