/**
 * Pure capability-driven lifecycle reducers for gameplay projections.
 *
 * Realtime event versions are hints only (Decision 9):
 * - ignore old/duplicate versions
 * - on gap, reconnect, or degraded realtime: mark needsSnapshotRepair and
 *   pause optimistic transitions
 * - applySnapshot replaces the projection and clears gap flags
 * - HTTP snapshots remain authoritative
 */

// ---------------------------------------------------------------------------
// Shared primitives
// ---------------------------------------------------------------------------

export type GamePhase =
  | "loading"
  | "playing"
  | "reveal"
  | "terminal"
  | "repairing";

export type RoomPhase =
  | "loading"
  | "lobby"
  | "playing"
  | "reveal"
  | "terminal"
  | "repairing";

export type QueuePhase =
  | "loading"
  | "idle"
  | "searching"
  | "matched"
  | "terminal"
  | "repairing";

export type PartyPhase =
  | "loading"
  | "lobby"
  | "ready"
  | "queued"
  | "matched"
  | "terminal"
  | "repairing";

export type MatchPhase =
  | "loading"
  | "playing"
  | "reveal"
  | "terminal"
  | "repairing";

export type Versioned = Readonly<{ version: number }>;

export type RepairFlags = Readonly<{
  needsSnapshotRepair: boolean;
  /** When true, optimistic UI transitions must not mutate domain fields. */
  optimisticPaused: boolean;
  lastError?: string;
}>;

type VersionDecision = "ignore" | "apply" | "gap";

/**
 * Compare a versioned hint against the current projection version.
 * Versions are hints: only the exact next version is applied.
 */
export function decideVersion(
  stateVersion: number,
  eventVersion: number,
): VersionDecision {
  if (eventVersion <= stateVersion) {
    return "ignore";
  }
  if (eventVersion === stateVersion + 1) {
    return "apply";
  }
  return "gap";
}

function withRepair<T extends RepairFlags & { phase: string }>(
  state: T,
  phase: T["phase"],
  lastError?: string,
): T {
  return {
    ...state,
    phase,
    needsSnapshotRepair: true,
    optimisticPaused: true,
    ...(lastError !== undefined ? { lastError } : {}),
  };
}

function clearRepairFields(): Pick<
  RepairFlags,
  "needsSnapshotRepair" | "optimisticPaused"
> & { lastError?: undefined } {
  return {
    needsSnapshotRepair: false,
    optimisticPaused: false,
    lastError: undefined,
  };
}

// ---------------------------------------------------------------------------
// Game (solo / quick / practice shell projection)
// ---------------------------------------------------------------------------

export type GameProjection = Readonly<{
  gameId: string;
  status: string;
  currentRoundNumber: number | null;
  totalScore: number;
  version: number;
  phase: GamePhase;
  needsSnapshotRepair: boolean;
  optimisticPaused: boolean;
  /** Local-only: guess in flight before authoritative ack. */
  guessPending: boolean;
  lastError?: string;
}>;

export type GameSnapshot = Readonly<{
  gameId: string;
  status: string;
  currentRoundNumber: number | null;
  totalScore: number;
  version: number;
  phase?: Exclude<GamePhase, "repairing" | "loading">;
}>;

export type GameEvent =
  | Readonly<{ type: "reconnect" }>
  | Readonly<{ type: "gap_detected"; reason?: string }>
  | Readonly<{ type: "degraded"; reason?: string }>
  | Readonly<{
      type: "versioned_hint";
      version: number;
      status?: string;
      currentRoundNumber?: number | null;
      totalScore?: number;
      phase?: Exclude<GamePhase, "repairing" | "loading">;
    }>
  | Readonly<{ type: "guess_submitted_optimistic" }>
  | Readonly<{
      type: "reveal";
      version: number;
      totalScore?: number;
      currentRoundNumber?: number | null;
    }>
  | Readonly<{
      type: "terminal";
      version: number;
      status?: string;
      totalScore?: number;
    }>
  | Readonly<{
      type: "round_advanced";
      version: number;
      currentRoundNumber: number;
      status?: string;
    }>;

export function createInitialGameProjection(
  gameId: string,
  version = 0,
): GameProjection {
  return {
    gameId,
    status: "loading",
    currentRoundNumber: null,
    totalScore: 0,
    version,
    phase: "loading",
    needsSnapshotRepair: false,
    optimisticPaused: false,
    guessPending: false,
  };
}

function gamePhaseFromSnapshot(snapshot: GameSnapshot): GamePhase {
  if (snapshot.phase) {
    return snapshot.phase;
  }
  const status = snapshot.status.toLowerCase();
  if (
    status === "completed" ||
    status === "ended" ||
    status === "cancelled" ||
    status === "forfeited"
  ) {
    return "terminal";
  }
  if (status === "reveal" || status === "revealing" || status === "round_result") {
    return "reveal";
  }
  if (status === "in_progress" || status === "active" || status === "playing") {
    return "playing";
  }
  return "playing";
}

export function applyGameSnapshot(
  _state: GameProjection,
  snapshot: GameSnapshot,
): GameProjection {
  return {
    gameId: snapshot.gameId,
    status: snapshot.status,
    currentRoundNumber: snapshot.currentRoundNumber,
    totalScore: snapshot.totalScore,
    version: snapshot.version,
    phase: gamePhaseFromSnapshot(snapshot),
    guessPending: false,
    ...clearRepairFields(),
  };
}

export function reduceGame(
  state: GameProjection,
  event: GameEvent,
): GameProjection {
  switch (event.type) {
    case "reconnect":
    case "gap_detected":
    case "degraded":
      return withRepair(
        state,
        "repairing",
        event.type === "reconnect"
          ? "reconnect"
          : event.type === "gap_detected"
            ? (event.reason ?? "version_gap")
            : (event.reason ?? "degraded"),
      );

    case "guess_submitted_optimistic":
      if (state.optimisticPaused || state.needsSnapshotRepair) {
        return state;
      }
      if (state.phase !== "playing") {
        return state;
      }
      return { ...state, guessPending: true };

    case "versioned_hint":
    case "reveal":
    case "terminal":
    case "round_advanced": {
      const decision = decideVersion(state.version, event.version);
      if (decision === "ignore") {
        return state;
      }
      if (decision === "gap") {
        return withRepair(state, "repairing", "version_gap");
      }

      // Contiguous next version — apply delta
      if (event.type === "reveal") {
        return {
          ...state,
          version: event.version,
          phase: "reveal",
          status: "reveal",
          guessPending: false,
          totalScore:
            event.totalScore !== undefined ? event.totalScore : state.totalScore,
          currentRoundNumber:
            event.currentRoundNumber !== undefined
              ? event.currentRoundNumber
              : state.currentRoundNumber,
        };
      }

      if (event.type === "terminal") {
        return {
          ...state,
          version: event.version,
          phase: "terminal",
          status: event.status ?? "completed",
          guessPending: false,
          totalScore:
            event.totalScore !== undefined ? event.totalScore : state.totalScore,
        };
      }

      if (event.type === "round_advanced") {
        return {
          ...state,
          version: event.version,
          phase: "playing",
          status: event.status ?? "in_progress",
          currentRoundNumber: event.currentRoundNumber,
          guessPending: false,
        };
      }

      // versioned_hint
      return {
        ...state,
        version: event.version,
        status: event.status ?? state.status,
        currentRoundNumber:
          event.currentRoundNumber !== undefined
            ? event.currentRoundNumber
            : state.currentRoundNumber,
        totalScore:
          event.totalScore !== undefined ? event.totalScore : state.totalScore,
        phase: event.phase ?? state.phase,
        guessPending: false,
      };
    }

    default: {
      const _exhaustive: never = event;
      return _exhaustive;
    }
  }
}

// ---------------------------------------------------------------------------
// Room (party lobby)
// ---------------------------------------------------------------------------

export type RoomProjection = Readonly<{
  roomCode: string;
  status: string;
  hostPlayerId: string | null;
  playerCount: number;
  readyCount: number;
  version: number;
  phase: RoomPhase;
  needsSnapshotRepair: boolean;
  optimisticPaused: boolean;
  lastError?: string;
}>;

export type RoomSnapshot = Readonly<{
  roomCode: string;
  status: string;
  hostPlayerId: string | null;
  playerCount: number;
  readyCount: number;
  version: number;
  phase?: Exclude<RoomPhase, "repairing" | "loading">;
}>;

export type RoomEvent =
  | Readonly<{ type: "reconnect" }>
  | Readonly<{ type: "gap_detected"; reason?: string }>
  | Readonly<{ type: "degraded"; reason?: string }>
  | Readonly<{
      type: "versioned_hint";
      version: number;
      status?: string;
      hostPlayerId?: string | null;
      playerCount?: number;
      readyCount?: number;
      phase?: Exclude<RoomPhase, "repairing" | "loading">;
    }>
  | Readonly<{
      type: "player_joined";
      version: number;
      playerCount: number;
    }>
  | Readonly<{
      type: "player_left";
      version: number;
      playerCount: number;
      hostPlayerId?: string | null;
    }>
  | Readonly<{
      type: "ready_changed";
      version: number;
      readyCount: number;
    }>
  | Readonly<{ type: "game_started"; version: number; status?: string }>
  | Readonly<{ type: "reveal"; version: number }>
  | Readonly<{ type: "terminal"; version: number; status?: string }>;

export function createInitialRoomProjection(
  roomCode: string,
  version = 0,
): RoomProjection {
  return {
    roomCode,
    status: "loading",
    hostPlayerId: null,
    playerCount: 0,
    readyCount: 0,
    version,
    phase: "loading",
    needsSnapshotRepair: false,
    optimisticPaused: false,
  };
}

function roomPhaseFromSnapshot(snapshot: RoomSnapshot): RoomPhase {
  if (snapshot.phase) {
    return snapshot.phase;
  }
  const status = snapshot.status.toLowerCase();
  if (
    status === "closed" ||
    status === "cancelled" ||
    status === "completed" ||
    status === "ended"
  ) {
    return "terminal";
  }
  if (status === "reveal" || status === "revealing") {
    return "reveal";
  }
  if (
    status === "in_progress" ||
    status === "active" ||
    status === "playing" ||
    status === "started"
  ) {
    return "playing";
  }
  return "lobby";
}

export function applyRoomSnapshot(
  _state: RoomProjection,
  snapshot: RoomSnapshot,
): RoomProjection {
  return {
    roomCode: snapshot.roomCode,
    status: snapshot.status,
    hostPlayerId: snapshot.hostPlayerId,
    playerCount: snapshot.playerCount,
    readyCount: snapshot.readyCount,
    version: snapshot.version,
    phase: roomPhaseFromSnapshot(snapshot),
    ...clearRepairFields(),
  };
}

export function reduceRoom(
  state: RoomProjection,
  event: RoomEvent,
): RoomProjection {
  switch (event.type) {
    case "reconnect":
    case "gap_detected":
    case "degraded":
      return withRepair(
        state,
        "repairing",
        event.type === "reconnect"
          ? "reconnect"
          : event.type === "gap_detected"
            ? (event.reason ?? "version_gap")
            : (event.reason ?? "degraded"),
      );

    case "versioned_hint":
    case "player_joined":
    case "player_left":
    case "ready_changed":
    case "game_started":
    case "reveal":
    case "terminal": {
      const decision = decideVersion(state.version, event.version);
      if (decision === "ignore") {
        return state;
      }
      if (decision === "gap") {
        return withRepair(state, "repairing", "version_gap");
      }

      if (event.type === "player_joined") {
        return {
          ...state,
          version: event.version,
          playerCount: event.playerCount,
          phase: state.phase === "loading" ? "lobby" : state.phase,
        };
      }

      if (event.type === "player_left") {
        return {
          ...state,
          version: event.version,
          playerCount: event.playerCount,
          hostPlayerId:
            event.hostPlayerId !== undefined
              ? event.hostPlayerId
              : state.hostPlayerId,
        };
      }

      if (event.type === "ready_changed") {
        return {
          ...state,
          version: event.version,
          readyCount: event.readyCount,
        };
      }

      if (event.type === "game_started") {
        return {
          ...state,
          version: event.version,
          phase: "playing",
          status: event.status ?? "in_progress",
        };
      }

      if (event.type === "reveal") {
        return {
          ...state,
          version: event.version,
          phase: "reveal",
          status: "reveal",
        };
      }

      if (event.type === "terminal") {
        return {
          ...state,
          version: event.version,
          phase: "terminal",
          status: event.status ?? "completed",
        };
      }

      return {
        ...state,
        version: event.version,
        status: event.status ?? state.status,
        hostPlayerId:
          event.hostPlayerId !== undefined
            ? event.hostPlayerId
            : state.hostPlayerId,
        playerCount:
          event.playerCount !== undefined
            ? event.playerCount
            : state.playerCount,
        readyCount:
          event.readyCount !== undefined ? event.readyCount : state.readyCount,
        phase: event.phase ?? state.phase,
      };
    }

    default: {
      const _exhaustive: never = event;
      return _exhaustive;
    }
  }
}

// ---------------------------------------------------------------------------
// Queue (matchmaking ticket projection)
// ---------------------------------------------------------------------------

export type QueueProjection = Readonly<{
  ticketId: string | null;
  status: string;
  playlist: string | null;
  format: string | null;
  matchId: string | null;
  version: number;
  phase: QueuePhase;
  needsSnapshotRepair: boolean;
  optimisticPaused: boolean;
  /** Local-only join attempt before authoritative ticket. */
  joinPending: boolean;
  lastError?: string;
}>;

export type QueueSnapshot = Readonly<{
  ticketId: string | null;
  status: string;
  playlist: string | null;
  format: string | null;
  matchId: string | null;
  version: number;
  phase?: Exclude<QueuePhase, "repairing" | "loading">;
}>;

export type QueueEvent =
  | Readonly<{ type: "reconnect" }>
  | Readonly<{ type: "gap_detected"; reason?: string }>
  | Readonly<{ type: "degraded"; reason?: string }>
  | Readonly<{ type: "join_optimistic"; playlist: string; format: string }>
  | Readonly<{
      type: "versioned_hint";
      version: number;
      status?: string;
      ticketId?: string | null;
      playlist?: string | null;
      format?: string | null;
      matchId?: string | null;
      phase?: Exclude<QueuePhase, "repairing" | "loading">;
    }>
  | Readonly<{
      type: "searching";
      version: number;
      ticketId: string;
      playlist?: string;
      format?: string;
    }>
  | Readonly<{
      type: "matched";
      version: number;
      matchId: string;
      ticketId?: string | null;
    }>
  | Readonly<{ type: "left"; version: number }>
  | Readonly<{ type: "terminal"; version: number; status?: string }>;

export function createInitialQueueProjection(version = 0): QueueProjection {
  return {
    ticketId: null,
    status: "not_queued",
    playlist: null,
    format: null,
    matchId: null,
    version,
    phase: "idle",
    needsSnapshotRepair: false,
    optimisticPaused: false,
    joinPending: false,
  };
}

function queuePhaseFromSnapshot(snapshot: QueueSnapshot): QueuePhase {
  if (snapshot.phase) {
    return snapshot.phase;
  }
  const status = snapshot.status.toLowerCase();
  if (status === "matched") {
    return "matched";
  }
  if (status === "searching" || status === "queued") {
    return "searching";
  }
  if (
    status === "cancelled" ||
    status === "expired" ||
    status === "failed" ||
    status === "left"
  ) {
    return "terminal";
  }
  return "idle";
}

export function applyQueueSnapshot(
  _state: QueueProjection,
  snapshot: QueueSnapshot,
): QueueProjection {
  return {
    ticketId: snapshot.ticketId,
    status: snapshot.status,
    playlist: snapshot.playlist,
    format: snapshot.format,
    matchId: snapshot.matchId,
    version: snapshot.version,
    phase: queuePhaseFromSnapshot(snapshot),
    joinPending: false,
    ...clearRepairFields(),
  };
}

export function reduceQueue(
  state: QueueProjection,
  event: QueueEvent,
): QueueProjection {
  switch (event.type) {
    case "reconnect":
    case "gap_detected":
    case "degraded":
      return withRepair(
        state,
        "repairing",
        event.type === "reconnect"
          ? "reconnect"
          : event.type === "gap_detected"
            ? (event.reason ?? "version_gap")
            : (event.reason ?? "degraded"),
      );

    case "join_optimistic":
      if (state.optimisticPaused || state.needsSnapshotRepair) {
        return state;
      }
      if (state.phase === "searching" || state.phase === "matched") {
        return state;
      }
      return {
        ...state,
        joinPending: true,
        playlist: event.playlist,
        format: event.format,
      };

    case "versioned_hint":
    case "searching":
    case "matched":
    case "left":
    case "terminal": {
      const decision = decideVersion(state.version, event.version);
      if (decision === "ignore") {
        return state;
      }
      if (decision === "gap") {
        return withRepair(state, "repairing", "version_gap");
      }

      if (event.type === "searching") {
        return {
          ...state,
          version: event.version,
          phase: "searching",
          status: "searching",
          ticketId: event.ticketId,
          playlist: event.playlist ?? state.playlist,
          format: event.format ?? state.format,
          joinPending: false,
        };
      }

      if (event.type === "matched") {
        return {
          ...state,
          version: event.version,
          phase: "matched",
          status: "matched",
          matchId: event.matchId,
          ticketId:
            event.ticketId !== undefined ? event.ticketId : state.ticketId,
          joinPending: false,
        };
      }

      if (event.type === "left") {
        return {
          ...state,
          version: event.version,
          phase: "idle",
          status: "not_queued",
          ticketId: null,
          matchId: null,
          joinPending: false,
        };
      }

      if (event.type === "terminal") {
        return {
          ...state,
          version: event.version,
          phase: "terminal",
          status: event.status ?? "cancelled",
          joinPending: false,
        };
      }

      return {
        ...state,
        version: event.version,
        status: event.status ?? state.status,
        ticketId:
          event.ticketId !== undefined ? event.ticketId : state.ticketId,
        playlist:
          event.playlist !== undefined ? event.playlist : state.playlist,
        format: event.format !== undefined ? event.format : state.format,
        matchId: event.matchId !== undefined ? event.matchId : state.matchId,
        phase: event.phase ?? state.phase,
        joinPending: false,
      };
    }

    default: {
      const _exhaustive: never = event;
      return _exhaustive;
    }
  }
}

// ---------------------------------------------------------------------------
// Party (premade matchmaking party)
// ---------------------------------------------------------------------------

export type PartyProjection = Readonly<{
  partyId: string;
  status: string;
  leaderId: string | null;
  memberCount: number;
  readyCount: number;
  version: number;
  phase: PartyPhase;
  needsSnapshotRepair: boolean;
  optimisticPaused: boolean;
  lastError?: string;
}>;

export type PartySnapshot = Readonly<{
  partyId: string;
  status: string;
  leaderId: string | null;
  memberCount: number;
  readyCount: number;
  version: number;
  phase?: Exclude<PartyPhase, "repairing" | "loading">;
}>;

export type PartyEvent =
  | Readonly<{ type: "reconnect" }>
  | Readonly<{ type: "gap_detected"; reason?: string }>
  | Readonly<{ type: "degraded"; reason?: string }>
  | Readonly<{
      type: "versioned_hint";
      version: number;
      status?: string;
      leaderId?: string | null;
      memberCount?: number;
      readyCount?: number;
      phase?: Exclude<PartyPhase, "repairing" | "loading">;
    }>
  | Readonly<{
      type: "member_updated";
      version: number;
      memberCount: number;
      readyCount?: number;
    }>
  | Readonly<{ type: "ready_changed"; version: number; readyCount: number }>
  | Readonly<{ type: "queued"; version: number }>
  | Readonly<{ type: "matched"; version: number }>
  | Readonly<{ type: "terminal"; version: number; status?: string }>;

export function createInitialPartyProjection(
  partyId: string,
  version = 0,
): PartyProjection {
  return {
    partyId,
    status: "loading",
    leaderId: null,
    memberCount: 0,
    readyCount: 0,
    version,
    phase: "loading",
    needsSnapshotRepair: false,
    optimisticPaused: false,
  };
}

function partyPhaseFromSnapshot(snapshot: PartySnapshot): PartyPhase {
  if (snapshot.phase) {
    return snapshot.phase;
  }
  const status = snapshot.status.toLowerCase();
  if (status === "matched") {
    return "matched";
  }
  if (status === "queued" || status === "searching") {
    return "queued";
  }
  if (status === "ready") {
    return "ready";
  }
  if (
    status === "disbanded" ||
    status === "cancelled" ||
    status === "completed"
  ) {
    return "terminal";
  }
  return "lobby";
}

export function applyPartySnapshot(
  _state: PartyProjection,
  snapshot: PartySnapshot,
): PartyProjection {
  return {
    partyId: snapshot.partyId,
    status: snapshot.status,
    leaderId: snapshot.leaderId,
    memberCount: snapshot.memberCount,
    readyCount: snapshot.readyCount,
    version: snapshot.version,
    phase: partyPhaseFromSnapshot(snapshot),
    ...clearRepairFields(),
  };
}

export function reduceParty(
  state: PartyProjection,
  event: PartyEvent,
): PartyProjection {
  switch (event.type) {
    case "reconnect":
    case "gap_detected":
    case "degraded":
      return withRepair(
        state,
        "repairing",
        event.type === "reconnect"
          ? "reconnect"
          : event.type === "gap_detected"
            ? (event.reason ?? "version_gap")
            : (event.reason ?? "degraded"),
      );

    case "versioned_hint":
    case "member_updated":
    case "ready_changed":
    case "queued":
    case "matched":
    case "terminal": {
      const decision = decideVersion(state.version, event.version);
      if (decision === "ignore") {
        return state;
      }
      if (decision === "gap") {
        return withRepair(state, "repairing", "version_gap");
      }

      if (event.type === "member_updated") {
        return {
          ...state,
          version: event.version,
          memberCount: event.memberCount,
          readyCount:
            event.readyCount !== undefined
              ? event.readyCount
              : state.readyCount,
          phase: state.phase === "loading" ? "lobby" : state.phase,
        };
      }

      if (event.type === "ready_changed") {
        return {
          ...state,
          version: event.version,
          readyCount: event.readyCount,
          phase:
            event.readyCount > 0 && state.phase === "lobby"
              ? "ready"
              : state.phase,
        };
      }

      if (event.type === "queued") {
        return {
          ...state,
          version: event.version,
          phase: "queued",
          status: "queued",
        };
      }

      if (event.type === "matched") {
        return {
          ...state,
          version: event.version,
          phase: "matched",
          status: "matched",
        };
      }

      if (event.type === "terminal") {
        return {
          ...state,
          version: event.version,
          phase: "terminal",
          status: event.status ?? "disbanded",
        };
      }

      return {
        ...state,
        version: event.version,
        status: event.status ?? state.status,
        leaderId:
          event.leaderId !== undefined ? event.leaderId : state.leaderId,
        memberCount:
          event.memberCount !== undefined
            ? event.memberCount
            : state.memberCount,
        readyCount:
          event.readyCount !== undefined ? event.readyCount : state.readyCount,
        phase: event.phase ?? state.phase,
      };
    }

    default: {
      const _exhaustive: never = event;
      return _exhaustive;
    }
  }
}

// ---------------------------------------------------------------------------
// Match (casual / ranked matchplay)
// ---------------------------------------------------------------------------

export type MatchProjection = Readonly<{
  matchId: string;
  status: string;
  currentRoundNumber: number | null;
  teamScore: number;
  opponentScore: number;
  version: number;
  phase: MatchPhase;
  needsSnapshotRepair: boolean;
  optimisticPaused: boolean;
  guessPending: boolean;
  lastError?: string;
}>;

export type MatchSnapshot = Readonly<{
  matchId: string;
  status: string;
  currentRoundNumber: number | null;
  teamScore: number;
  opponentScore: number;
  version: number;
  phase?: Exclude<MatchPhase, "repairing" | "loading">;
}>;

export type MatchEvent =
  | Readonly<{ type: "reconnect" }>
  | Readonly<{ type: "gap_detected"; reason?: string }>
  | Readonly<{ type: "degraded"; reason?: string }>
  | Readonly<{ type: "guess_submitted_optimistic" }>
  | Readonly<{
      type: "versioned_hint";
      version: number;
      status?: string;
      currentRoundNumber?: number | null;
      teamScore?: number;
      opponentScore?: number;
      phase?: Exclude<MatchPhase, "repairing" | "loading">;
    }>
  | Readonly<{
      type: "reveal";
      version: number;
      teamScore?: number;
      opponentScore?: number;
    }>
  | Readonly<{
      type: "round_advanced";
      version: number;
      currentRoundNumber: number;
    }>
  | Readonly<{
      type: "terminal";
      version: number;
      status?: string;
      teamScore?: number;
      opponentScore?: number;
    }>;

export function createInitialMatchProjection(
  matchId: string,
  version = 0,
): MatchProjection {
  return {
    matchId,
    status: "loading",
    currentRoundNumber: null,
    teamScore: 0,
    opponentScore: 0,
    version,
    phase: "loading",
    needsSnapshotRepair: false,
    optimisticPaused: false,
    guessPending: false,
  };
}

function matchPhaseFromSnapshot(snapshot: MatchSnapshot): MatchPhase {
  if (snapshot.phase) {
    return snapshot.phase;
  }
  const status = snapshot.status.toLowerCase();
  if (
    status === "completed" ||
    status === "ended" ||
    status === "forfeited" ||
    status === "cancelled"
  ) {
    return "terminal";
  }
  if (status === "reveal" || status === "revealing" || status === "round_result") {
    return "reveal";
  }
  return "playing";
}

export function applyMatchSnapshot(
  _state: MatchProjection,
  snapshot: MatchSnapshot,
): MatchProjection {
  return {
    matchId: snapshot.matchId,
    status: snapshot.status,
    currentRoundNumber: snapshot.currentRoundNumber,
    teamScore: snapshot.teamScore,
    opponentScore: snapshot.opponentScore,
    version: snapshot.version,
    phase: matchPhaseFromSnapshot(snapshot),
    guessPending: false,
    ...clearRepairFields(),
  };
}

export function reduceMatch(
  state: MatchProjection,
  event: MatchEvent,
): MatchProjection {
  switch (event.type) {
    case "reconnect":
    case "gap_detected":
    case "degraded":
      return withRepair(
        state,
        "repairing",
        event.type === "reconnect"
          ? "reconnect"
          : event.type === "gap_detected"
            ? (event.reason ?? "version_gap")
            : (event.reason ?? "degraded"),
      );

    case "guess_submitted_optimistic":
      if (state.optimisticPaused || state.needsSnapshotRepair) {
        return state;
      }
      if (state.phase !== "playing") {
        return state;
      }
      return { ...state, guessPending: true };

    case "versioned_hint":
    case "reveal":
    case "round_advanced":
    case "terminal": {
      const decision = decideVersion(state.version, event.version);
      if (decision === "ignore") {
        return state;
      }
      if (decision === "gap") {
        return withRepair(state, "repairing", "version_gap");
      }

      if (event.type === "reveal") {
        return {
          ...state,
          version: event.version,
          phase: "reveal",
          status: "reveal",
          guessPending: false,
          teamScore:
            event.teamScore !== undefined ? event.teamScore : state.teamScore,
          opponentScore:
            event.opponentScore !== undefined
              ? event.opponentScore
              : state.opponentScore,
        };
      }

      if (event.type === "round_advanced") {
        return {
          ...state,
          version: event.version,
          phase: "playing",
          status: "in_progress",
          currentRoundNumber: event.currentRoundNumber,
          guessPending: false,
        };
      }

      if (event.type === "terminal") {
        return {
          ...state,
          version: event.version,
          phase: "terminal",
          status: event.status ?? "completed",
          guessPending: false,
          teamScore:
            event.teamScore !== undefined ? event.teamScore : state.teamScore,
          opponentScore:
            event.opponentScore !== undefined
              ? event.opponentScore
              : state.opponentScore,
        };
      }

      return {
        ...state,
        version: event.version,
        status: event.status ?? state.status,
        currentRoundNumber:
          event.currentRoundNumber !== undefined
            ? event.currentRoundNumber
            : state.currentRoundNumber,
        teamScore:
          event.teamScore !== undefined ? event.teamScore : state.teamScore,
        opponentScore:
          event.opponentScore !== undefined
            ? event.opponentScore
            : state.opponentScore,
        phase: event.phase ?? state.phase,
        guessPending: false,
      };
    }

    default: {
      const _exhaustive: never = event;
      return _exhaustive;
    }
  }
}
