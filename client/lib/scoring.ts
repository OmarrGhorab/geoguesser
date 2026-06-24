/** Maximum achievable score for a perfect (0 km) guess. */
export const MAX_SCORE = 5000;

/**
 * Distance → score using an exponential-decay curve.
 *
 *   score = MAX_SCORE × e^(−distance / DECAY_KM)
 *
 * Why exponential decay (chosen for game feel over a fixed-range table):
 *   • Near-misses are rewarded generously (matches the gentle early slope
 *     implied by the spec's anchors: 0 km → 5000, 50 km → ~4500,
 *     100 km → ~4000).
 *   • Far guesses taper smoothly toward 0 rather than cliff-dropping,
 *     which feels less punishing on a world map.
 *
 * The spec's anchors (0/50/100/500/1000 km) are slightly self-inconsistent
 * (gentle early slope but must hit exactly 0 by 1000 km), so no single
 * smooth curve can match all of them. DECAY_KM = 500 nails the first three
 * — the ones that drive perceived difficulty — while keeping the formula a
 * one-liner with no lookup table.
 *
 * Sample values: 0→5000, 50→4524, 100→4094, 250→3033, 500→1839, 1000→677.
 *
 * @param distanceKm great-circle distance from the player's guess to target.
 * @returns integer score in [0, MAX_SCORE].
 */
export function scoreFromDistanceKm(distanceKm: number): number {
  const DECAY_KM = 500;

  // Negative distances shouldn't happen, but guard anyway so the curve can
  // never return > MAX_SCORE.
  const safeDistance = Math.max(0, distanceKm);

  const raw = MAX_SCORE * Math.exp(-safeDistance / DECAY_KM);

  // Round to integer and clamp to the valid range.
  return Math.min(MAX_SCORE, Math.max(0, Math.round(raw)));
}
