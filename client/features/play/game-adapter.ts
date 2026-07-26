export type PlayCapabilityMode =
  | "solo"
  | "quick_play"
  | "practice"
  | "daily";

export type GameCapabilities = Readonly<{
  mode: PlayCapabilityMode;
  /** Round has a server deadline the UI should countdown. */
  timed: boolean;
  /** Practice-style sessions that append rounds on demand. */
  openEnded: boolean;
  /** Multiplayer shared reveal (false for solo-family modes). */
  sharedReveal: boolean;
  /** Show fixed round progress (e.g. 3/5) rather than open practice count. */
  showProgression: boolean;
  /** Client may POST timeout after server deadline. */
  allowTimeout: boolean;
  /** Finite round_count owned by the game row. */
  fixedRounds: boolean;
  /** Immediate answer reveal after each guess. */
  immediateReveal: boolean;
  /** Neutral scoring — no ranked/competitive progression. */
  neutralProgression: boolean;
  /** Cursor-paged practice history is available. */
  history: boolean;
  /** Explicit end-session command is required. */
  explicitEnd: boolean;
  /** Server owns map/timer defaults (Quick Play). */
  serverDefaults: boolean;
}>;

const SOLO: GameCapabilities = {
  mode: "solo",
  timed: true,
  openEnded: false,
  sharedReveal: false,
  showProgression: true,
  allowTimeout: true,
  fixedRounds: true,
  immediateReveal: true,
  neutralProgression: true,
  history: false,
  explicitEnd: false,
  serverDefaults: false,
};

const QUICK_PLAY: GameCapabilities = {
  mode: "quick_play",
  timed: true,
  openEnded: false,
  sharedReveal: false,
  showProgression: true,
  allowTimeout: true,
  fixedRounds: true,
  immediateReveal: true,
  neutralProgression: true,
  history: false,
  explicitEnd: false,
  serverDefaults: true,
};

const PRACTICE: GameCapabilities = {
  mode: "practice",
  timed: false,
  openEnded: true,
  sharedReveal: false,
  showProgression: false,
  allowTimeout: false,
  fixedRounds: false,
  immediateReveal: true,
  neutralProgression: true,
  history: true,
  explicitEnd: true,
  serverDefaults: false,
};

const DAILY: GameCapabilities = {
  mode: "daily",
  timed: true,
  openEnded: false,
  sharedReveal: false,
  showProgression: true,
  allowTimeout: true,
  fixedRounds: true,
  immediateReveal: true,
  neutralProgression: true,
  history: false,
  explicitEnd: false,
  serverDefaults: true,
};

const BY_MODE: Record<PlayCapabilityMode, GameCapabilities> = {
  solo: SOLO,
  quick_play: QUICK_PLAY,
  practice: PRACTICE,
  daily: DAILY,
};

/** Pure capability object for solo-family gameplay shells. */
export function getGameCapabilities(mode: PlayCapabilityMode): GameCapabilities {
  return BY_MODE[mode];
}

/**
 * Solo may be configured without a timer. Prefer explicit timer presence when
 * known; fall back to the mode default when only the mode string is available.
 */
export function resolveSoloCapabilities(options?: {
  timerSeconds?: number | null;
}): GameCapabilities {
  const timed =
    options?.timerSeconds === undefined
      ? true
      : options.timerSeconds != null && options.timerSeconds > 0;
  return {
    ...SOLO,
    timed,
    allowTimeout: timed,
  };
}
