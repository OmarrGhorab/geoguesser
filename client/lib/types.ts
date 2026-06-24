/**
 * Shared domain types used across the game.
 * Keeping them here keeps page.tsx and the components decoupled from the
 * specific data shape and from each other.
 */

/** A geographic point. Matches the Google Maps `{ lat, lng }` convention. */
export type LatLng = {
  lat: number;
  lng: number;
};

/** Two distinct phases the game UI can be in. */
export type GamePhase = "guessing" | "revealed";

/** Result of a single round, surfaced to the ResultCard. */
export type GuessResult = {
  /** Great-circle distance between guess and target, in kilometers. */
  distanceKm: number;
  /** Points earned this round, 0–5000. */
  score: number;
  /** Where the player clicked. */
  guess: LatLng;
  /** The landmark's true coordinates. */
  actual: LatLng;
};
