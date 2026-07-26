/**
 * Canonical player-facing game-mode registry for feature 014-game-mode-pages.
 *
 * Exactly eleven selectable modes. Legacy aliases are recoverable only via
 * normalizeMode and never appear in listSelectableModes().
 */

export const CANONICAL_MODES = [
  "solo",
  "practice",
  "daily",
  "quick_play",
  "party_lobby",
  "casual_solo",
  "casual_duo",
  "casual_squad",
  "ranked_solo",
  "ranked_duo",
  "ranked_squad",
] as const;

export type CanonicalMode = (typeof CANONICAL_MODES)[number];

/** One-way recovery aliases — never selectable product choices. */
export const LEGACY_ALIASES = {
  private_room: "party_lobby",
  ranked_standard: "ranked_solo",
  /** Context-dependent in product recovery; product default maps to ranked_solo. */
  ranked: "ranked_solo",
} as const satisfies Record<string, CanonicalMode>;

export type LegacyAlias = keyof typeof LEGACY_ALIASES;

export type SessionRequirement = "guest_or_account" | "account";

export type TeamSize = 1 | 2 | 4;

export type ModeCapabilities = Readonly<{
  timed: boolean;
  openEnded: boolean;
  sharedReveal: boolean;
  teams: boolean;
  speedBonus: boolean;
  progression: boolean;
  chat: boolean;
  hostControls: boolean;
}>;

export type ModeDefinition = Readonly<{
  mode: CanonicalMode;
  /** next-intl message key, e.g. Play.modes.solo */
  messageKey: `Play.modes.${CanonicalMode}`;
  /** next-intl description key — no "." in the leaf name (next-intl nesting). */
  descriptionKey: `Play.modes.${CanonicalMode}Description`;
  sessionRequirement: SessionRequirement;
  teamSize: TeamSize;
  capabilities: ModeCapabilities;
}>;

const CAPS = {
  solo: {
    timed: true,
    openEnded: false,
    sharedReveal: false,
    teams: false,
    speedBonus: false,
    progression: true,
    chat: false,
    hostControls: false,
  },
  practice: {
    timed: false,
    openEnded: true,
    sharedReveal: false,
    teams: false,
    speedBonus: false,
    progression: false,
    chat: false,
    hostControls: false,
  },
  daily: {
    timed: true,
    openEnded: false,
    sharedReveal: false,
    teams: false,
    speedBonus: false,
    progression: true,
    chat: false,
    hostControls: false,
  },
  quick_play: {
    timed: true,
    openEnded: false,
    sharedReveal: false,
    teams: false,
    speedBonus: false,
    progression: true,
    chat: false,
    hostControls: false,
  },
  party_lobby: {
    timed: true,
    openEnded: false,
    sharedReveal: true,
    teams: false,
    speedBonus: false,
    progression: false,
    chat: false,
    hostControls: true,
  },
  casual_solo: {
    timed: false,
    openEnded: false,
    sharedReveal: true,
    teams: false,
    speedBonus: false,
    progression: false,
    chat: false,
    hostControls: false,
  },
  casual_duo: {
    timed: false,
    openEnded: false,
    sharedReveal: true,
    teams: true,
    speedBonus: false,
    progression: false,
    chat: true,
    hostControls: false,
  },
  casual_squad: {
    timed: false,
    openEnded: false,
    sharedReveal: true,
    teams: true,
    speedBonus: false,
    progression: false,
    chat: true,
    hostControls: false,
  },
  ranked_solo: {
    timed: true,
    openEnded: false,
    sharedReveal: true,
    teams: false,
    speedBonus: true,
    progression: true,
    chat: false,
    hostControls: false,
  },
  ranked_duo: {
    timed: true,
    openEnded: false,
    sharedReveal: true,
    teams: true,
    speedBonus: true,
    progression: true,
    chat: true,
    hostControls: false,
  },
  ranked_squad: {
    timed: true,
    openEnded: false,
    sharedReveal: true,
    teams: true,
    speedBonus: true,
    progression: true,
    chat: true,
    hostControls: false,
  },
} as const satisfies Record<CanonicalMode, ModeCapabilities>;

const SESSION: Record<CanonicalMode, SessionRequirement> = {
  solo: "guest_or_account",
  practice: "guest_or_account",
  daily: "guest_or_account",
  quick_play: "guest_or_account",
  party_lobby: "guest_or_account",
  casual_solo: "account",
  casual_duo: "account",
  casual_squad: "account",
  ranked_solo: "account",
  ranked_duo: "account",
  ranked_squad: "account",
};

const TEAM_SIZE: Record<CanonicalMode, TeamSize> = {
  solo: 1,
  practice: 1,
  daily: 1,
  quick_play: 1,
  party_lobby: 1,
  casual_solo: 1,
  casual_duo: 2,
  casual_squad: 4,
  ranked_solo: 1,
  ranked_duo: 2,
  ranked_squad: 4,
};

function definitionFor(mode: CanonicalMode): ModeDefinition {
  return {
    mode,
    messageKey: `Play.modes.${mode}`,
    descriptionKey: `Play.modes.${mode}Description`,
    sessionRequirement: SESSION[mode],
    teamSize: TEAM_SIZE[mode],
    capabilities: CAPS[mode],
  };
}

const MODE_DEFINITIONS: Record<CanonicalMode, ModeDefinition> =
  Object.fromEntries(
    CANONICAL_MODES.map((mode) => [mode, definitionFor(mode)]),
  ) as Record<CanonicalMode, ModeDefinition>;

const CANONICAL_MODE_SET: ReadonlySet<string> = new Set(CANONICAL_MODES);

export function isCanonicalMode(value: unknown): value is CanonicalMode {
  return typeof value === "string" && CANONICAL_MODE_SET.has(value);
}

/**
 * Normalizes inbound mode strings at recovery boundaries only.
 * Canonical values pass through; legacy aliases map one-way; unknown → null.
 */
export function normalizeMode(input: unknown): CanonicalMode | null {
  if (typeof input !== "string" || input.length === 0) {
    return null;
  }

  if (isCanonicalMode(input)) {
    return input;
  }

  if (Object.hasOwn(LEGACY_ALIASES, input)) {
    return LEGACY_ALIASES[input as LegacyAlias];
  }

  return null;
}

export function getModeDefinition(mode: CanonicalMode): ModeDefinition {
  return MODE_DEFINITIONS[mode];
}

/** Exactly the eleven player-facing choices; never includes legacy aliases. */
export function listSelectableModes(): readonly ModeDefinition[] {
  return CANONICAL_MODES.map((mode) => MODE_DEFINITIONS[mode]);
}
