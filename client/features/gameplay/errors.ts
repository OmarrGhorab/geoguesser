/**
 * Stable gameplay API error categories and recovery actions.
 * Message keys resolve under next-intl namespace `gameplay.errors.*`.
 */

export const GAMEPLAY_ERROR_CATEGORIES = [
  "unauthorized",
  "forbidden",
  "not_found",
  "conflict",
  "stale_version",
  "validation",
  "rate_limited",
  "unavailable",
  "wrong_mode",
  "idempotency_conflict",
  "host_action_required",
  "progression_pending",
  "already_active",
  "already_matched",
  "full",
  "expired",
  "cancelled",
  "round_not_revealed",
  "ineligible",
  "internal",
] as const;

export type GameplayErrorCategory = (typeof GAMEPLAY_ERROR_CATEGORIES)[number];

export const RECOVERY_ACTIONS = [
  "retry",
  "login",
  "exit",
  "refresh",
  "wait",
  "none",
] as const;

export type RecoveryAction = (typeof RECOVERY_ACTIONS)[number];

const CODE_TO_CATEGORY: Record<string, GameplayErrorCategory> = {
  unauthorized: "unauthorized",
  unauthenticated: "unauthorized",
  session_required: "unauthorized",
  session_expired: "unauthorized",

  forbidden: "forbidden",
  csrf_failed: "forbidden",
  account_ineligible: "ineligible",
  ineligible: "ineligible",
  active_party_conflict: "conflict",

  not_found: "not_found",
  game_not_found: "not_found",
  room_not_found: "not_found",
  party_not_found: "not_found",
  match_not_found: "not_found",
  map_not_found: "not_found",

  conflict: "conflict",
  stale_version: "stale_version",
  version_conflict: "stale_version",
  party_version_mismatch: "stale_version",

  validation: "validation",
  validation_failed: "validation",
  invalid_request: "validation",
  bad_request: "validation",

  rate_limited: "rate_limited",
  too_many_requests: "rate_limited",

  unavailable: "unavailable",
  temporarily_unavailable: "unavailable",
  dependency_unavailable: "unavailable",
  image_service_unavailable: "unavailable",
  service_unavailable: "unavailable",

  wrong_mode: "wrong_mode",
  wrong_game_mode: "wrong_mode",

  idempotency_conflict: "idempotency_conflict",

  host_action_required: "host_action_required",

  progression_pending: "progression_pending",

  already_active: "already_active",
  already_guessed: "conflict",
  already_matched: "already_matched",
  already_queued: "already_matched",

  full: "full",
  room_full: "full",
  party_full: "full",

  expired: "expired",
  room_expired: "expired",
  invite_expired: "expired",

  cancelled: "cancelled",
  room_cancelled: "cancelled",

  round_not_revealed: "round_not_revealed",
  current_round_incomplete: "conflict",
  round_closed: "conflict",
  game_not_active: "conflict",
  game_not_completed: "conflict",
  not_enough_locations: "unavailable",

  internal: "internal",
  internal_error: "internal",
};

/**
 * Map an HTTP status + backend error code to a stable gameplay category.
 * Prefer explicit codes; fall back to status when the code is unknown.
 */
export function mapApiErrorToGameplayCategory(
  status: number,
  code?: string | null,
): GameplayErrorCategory {
  const normalized = (code ?? "").trim().toLowerCase();
  if (normalized && CODE_TO_CATEGORY[normalized]) {
    return CODE_TO_CATEGORY[normalized];
  }

  switch (status) {
    case 401:
      return "unauthorized";
    case 403:
      return "forbidden";
    case 404:
      return "not_found";
    case 409:
      return "conflict";
    case 422:
      return "validation";
    case 429:
      return "rate_limited";
    case 202:
      return "progression_pending";
    case 503:
    case 502:
    case 504:
      return "unavailable";
    case 400:
      return "validation";
    default:
      if (status >= 500) return "internal";
      return "internal";
  }
}

const CATEGORY_RECOVERY: Record<GameplayErrorCategory, RecoveryAction> = {
  unauthorized: "login",
  forbidden: "exit",
  not_found: "exit",
  conflict: "refresh",
  stale_version: "refresh",
  validation: "none",
  rate_limited: "wait",
  unavailable: "retry",
  wrong_mode: "exit",
  idempotency_conflict: "refresh",
  host_action_required: "none",
  progression_pending: "wait",
  already_active: "refresh",
  already_matched: "refresh",
  full: "exit",
  expired: "exit",
  cancelled: "exit",
  round_not_revealed: "wait",
  ineligible: "exit",
  internal: "retry",
};

export function recoveryActionForCategory(
  category: GameplayErrorCategory,
): RecoveryAction {
  return CATEGORY_RECOVERY[category];
}

/**
 * Leaf key under the next-intl `gameplay.errors` namespace.
 * Example: t(`errors.${messageKeyForCategory(category)}`) within gameplay scope,
 * or full path `gameplay.errors.${key}` from root.
 */
export function messageKeyForCategory(
  category: GameplayErrorCategory,
): GameplayErrorCategory {
  return category;
}

/** Full message path for catalogs that nest under `gameplay`. */
export function gameplayErrorMessagePath(
  category: GameplayErrorCategory,
): `gameplay.errors.${GameplayErrorCategory}` {
  return `gameplay.errors.${category}`;
}

export function mapApiErrorToGameplay(
  status: number,
  code?: string | null,
): {
  category: GameplayErrorCategory;
  recovery: RecoveryAction;
  messageKey: GameplayErrorCategory;
  messagePath: `gameplay.errors.${GameplayErrorCategory}`;
} {
  const category = mapApiErrorToGameplayCategory(status, code);
  return {
    category,
    recovery: recoveryActionForCategory(category),
    messageKey: messageKeyForCategory(category),
    messagePath: gameplayErrorMessagePath(category),
  };
}
