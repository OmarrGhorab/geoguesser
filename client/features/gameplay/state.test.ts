import { describe, expect, it } from "vitest";
import {
  applyGameSnapshot,
  applyMatchSnapshot,
  applyPartySnapshot,
  applyQueueSnapshot,
  applyRoomSnapshot,
  createInitialGameProjection,
  createInitialMatchProjection,
  createInitialPartyProjection,
  createInitialQueueProjection,
  createInitialRoomProjection,
  decideVersion,
  reduceGame,
  reduceMatch,
  reduceParty,
  reduceQueue,
  reduceRoom,
  type GameProjection,
  type GameSnapshot,
  type MatchProjection,
  type MatchSnapshot,
  type PartyProjection,
  type PartySnapshot,
  type QueueProjection,
  type QueueSnapshot,
  type RoomProjection,
  type RoomSnapshot,
} from "./state";

// ---------------------------------------------------------------------------
// Shared version decision
// ---------------------------------------------------------------------------

describe("decideVersion", () => {
  it("ignores older versions", () => {
    expect(decideVersion(5, 3)).toBe("ignore");
    expect(decideVersion(5, 4)).toBe("ignore");
  });

  it("ignores duplicate versions", () => {
    expect(decideVersion(5, 5)).toBe("ignore");
  });

  it("applies the exact next version", () => {
    expect(decideVersion(5, 6)).toBe("apply");
    expect(decideVersion(0, 1)).toBe("apply");
  });

  it("flags a gap when version jumps more than one", () => {
    expect(decideVersion(5, 7)).toBe("gap");
    expect(decideVersion(1, 10)).toBe("gap");
  });
});

// ---------------------------------------------------------------------------
// Game projection
// ---------------------------------------------------------------------------

describe("reduceGame / applyGameSnapshot", () => {
  function hydratedGame(overrides: Partial<GameSnapshot> = {}): GameProjection {
    return applyGameSnapshot(createInitialGameProjection("game-1"), {
      gameId: "game-1",
      status: "in_progress",
      currentRoundNumber: 1,
      totalScore: 0,
      version: 3,
      phase: "playing",
      ...overrides,
    });
  }

  it("applies a contiguous versioned hint", () => {
    const state = hydratedGame();
    const next = reduceGame(state, {
      type: "versioned_hint",
      version: 4,
      totalScore: 1200,
      currentRoundNumber: 2,
      phase: "playing",
    });
    expect(next.version).toBe(4);
    expect(next.totalScore).toBe(1200);
    expect(next.currentRoundNumber).toBe(2);
    expect(next.needsSnapshotRepair).toBe(false);
    expect(next.phase).toBe("playing");
  });

  it("ignores old and duplicate versions", () => {
    const state = hydratedGame({ version: 5, totalScore: 900 });
    const older = reduceGame(state, {
      type: "versioned_hint",
      version: 3,
      totalScore: 0,
    });
    const same = reduceGame(state, {
      type: "versioned_hint",
      version: 5,
      totalScore: 0,
    });
    expect(older).toBe(state);
    expect(same).toBe(state);
  });

  it("marks repair on version gap and pauses optimistic transitions", () => {
    const state = hydratedGame({ version: 2 });
    const gapped = reduceGame(state, {
      type: "versioned_hint",
      version: 5,
      totalScore: 9999,
    });
    expect(gapped.needsSnapshotRepair).toBe(true);
    expect(gapped.optimisticPaused).toBe(true);
    expect(gapped.phase).toBe("repairing");
    expect(gapped.totalScore).toBe(0);
    expect(gapped.version).toBe(2);
    expect(gapped.lastError).toBe("version_gap");

    const optimistic = reduceGame(gapped, {
      type: "guess_submitted_optimistic",
    });
    expect(optimistic.guessPending).toBe(false);
    expect(optimistic).toBe(gapped);
  });

  it("marks repair on reconnect and degraded realtime", () => {
    const state = hydratedGame();
    const reconnected = reduceGame(state, { type: "reconnect" });
    expect(reconnected.needsSnapshotRepair).toBe(true);
    expect(reconnected.optimisticPaused).toBe(true);
    expect(reconnected.phase).toBe("repairing");
    expect(reconnected.lastError).toBe("reconnect");

    const degraded = reduceGame(state, {
      type: "degraded",
      reason: "channel_unstable",
    });
    expect(degraded.needsSnapshotRepair).toBe(true);
    expect(degraded.lastError).toBe("channel_unstable");
  });

  it("allows optimistic guess only while playing and not repairing", () => {
    const playing = hydratedGame();
    const pending = reduceGame(playing, { type: "guess_submitted_optimistic" });
    expect(pending.guessPending).toBe(true);

    const reveal = reduceGame(playing, {
      type: "reveal",
      version: 4,
      totalScore: 500,
    });
    expect(reveal.phase).toBe("reveal");
    expect(reveal.guessPending).toBe(false);
    const blocked = reduceGame(reveal, { type: "guess_submitted_optimistic" });
    expect(blocked.guessPending).toBe(false);
  });

  it("applies reveal and terminal deltas at next version", () => {
    const state = hydratedGame({ version: 3, totalScore: 100 });
    const revealed = reduceGame(state, {
      type: "reveal",
      version: 4,
      totalScore: 2500,
    });
    expect(revealed.phase).toBe("reveal");
    expect(revealed.totalScore).toBe(2500);
    expect(revealed.version).toBe(4);

    const terminal = reduceGame(revealed, {
      type: "terminal",
      version: 5,
      status: "completed",
      totalScore: 4800,
    });
    expect(terminal.phase).toBe("terminal");
    expect(terminal.status).toBe("completed");
    expect(terminal.totalScore).toBe(4800);
  });

  it("advances rounds contiguously", () => {
    const state = hydratedGame({ version: 4, currentRoundNumber: 1 });
    const next = reduceGame(state, {
      type: "round_advanced",
      version: 5,
      currentRoundNumber: 2,
    });
    expect(next.currentRoundNumber).toBe(2);
    expect(next.phase).toBe("playing");
    expect(next.guessPending).toBe(false);
  });

  it("HTTP snapshot always wins and clears repair flags", () => {
    const repairing = reduceGame(hydratedGame({ version: 2 }), {
      type: "gap_detected",
    });
    expect(repairing.needsSnapshotRepair).toBe(true);

    const repaired = applyGameSnapshot(repairing, {
      gameId: "game-1",
      status: "in_progress",
      currentRoundNumber: 3,
      totalScore: 3100,
      version: 8,
      phase: "playing",
    });
    expect(repaired.version).toBe(8);
    expect(repaired.totalScore).toBe(3100);
    expect(repaired.currentRoundNumber).toBe(3);
    expect(repaired.needsSnapshotRepair).toBe(false);
    expect(repaired.optimisticPaused).toBe(false);
    expect(repaired.phase).toBe("playing");
    expect(repaired.guessPending).toBe(false);
    expect(repaired.lastError).toBeUndefined();
  });

  it("derives terminal phase from snapshot status when phase omitted", () => {
    const state = createInitialGameProjection("game-1");
    const done = applyGameSnapshot(state, {
      gameId: "game-1",
      status: "completed",
      currentRoundNumber: 5,
      totalScore: 9999,
      version: 12,
    });
    expect(done.phase).toBe("terminal");
  });
});

// ---------------------------------------------------------------------------
// Room projection
// ---------------------------------------------------------------------------

describe("reduceRoom / applyRoomSnapshot", () => {
  function hydratedRoom(overrides: Partial<RoomSnapshot> = {}): RoomProjection {
    return applyRoomSnapshot(createInitialRoomProjection("ABCD"), {
      roomCode: "ABCD",
      status: "lobby",
      hostPlayerId: "host-1",
      playerCount: 2,
      readyCount: 1,
      version: 4,
      phase: "lobby",
      ...overrides,
    });
  }

  it("applies contiguous lobby deltas", () => {
    const state = hydratedRoom();
    const joined = reduceRoom(state, {
      type: "player_joined",
      version: 5,
      playerCount: 3,
    });
    expect(joined.playerCount).toBe(3);
    expect(joined.version).toBe(5);

    const ready = reduceRoom(joined, {
      type: "ready_changed",
      version: 6,
      readyCount: 3,
    });
    expect(ready.readyCount).toBe(3);
  });

  it("ignores old/duplicate room events", () => {
    const state = hydratedRoom({ version: 10, playerCount: 4 });
    expect(
      reduceRoom(state, {
        type: "player_joined",
        version: 9,
        playerCount: 99,
      }),
    ).toBe(state);
    expect(
      reduceRoom(state, {
        type: "player_joined",
        version: 10,
        playerCount: 99,
      }),
    ).toBe(state);
  });

  it("repairs on gap and reconnect", () => {
    const state = hydratedRoom({ version: 1 });
    const gap = reduceRoom(state, {
      type: "versioned_hint",
      version: 4,
      playerCount: 50,
    });
    expect(gap.needsSnapshotRepair).toBe(true);
    expect(gap.phase).toBe("repairing");
    expect(gap.playerCount).toBe(2);

    const reconnect = reduceRoom(state, { type: "reconnect" });
    expect(reconnect.needsSnapshotRepair).toBe(true);
    expect(reconnect.optimisticPaused).toBe(true);
  });

  it("transitions through game_started, reveal, and terminal", () => {
    let state = hydratedRoom({ version: 4 });
    state = reduceRoom(state, {
      type: "game_started",
      version: 5,
      status: "in_progress",
    });
    expect(state.phase).toBe("playing");

    state = reduceRoom(state, { type: "reveal", version: 6 });
    expect(state.phase).toBe("reveal");

    state = reduceRoom(state, {
      type: "terminal",
      version: 7,
      status: "completed",
    });
    expect(state.phase).toBe("terminal");
    expect(state.status).toBe("completed");
  });

  it("snapshot replaces projection and clears gap flags", () => {
    const broken = reduceRoom(hydratedRoom({ version: 2 }), {
      type: "gap_detected",
      reason: "missed_events",
    });
    const fixed = applyRoomSnapshot(broken, {
      roomCode: "ABCD",
      status: "in_progress",
      hostPlayerId: "host-2",
      playerCount: 8,
      readyCount: 8,
      version: 20,
      phase: "playing",
    });
    expect(fixed.version).toBe(20);
    expect(fixed.playerCount).toBe(8);
    expect(fixed.hostPlayerId).toBe("host-2");
    expect(fixed.needsSnapshotRepair).toBe(false);
    expect(fixed.optimisticPaused).toBe(false);
    expect(fixed.phase).toBe("playing");
  });
});

// ---------------------------------------------------------------------------
// Queue projection
// ---------------------------------------------------------------------------

describe("reduceQueue / applyQueueSnapshot", () => {
  function searchingQueue(
    overrides: Partial<QueueSnapshot> = {},
  ): QueueProjection {
    return applyQueueSnapshot(createInitialQueueProjection(0), {
      ticketId: "ticket-1",
      status: "searching",
      playlist: "casual",
      format: "solo",
      matchId: null,
      version: 2,
      phase: "searching",
      ...overrides,
    });
  }

  it("applies searching and matched contiguous events", () => {
    const idle = createInitialQueueProjection(0);
    const searching = reduceQueue(idle, {
      type: "searching",
      version: 1,
      ticketId: "ticket-1",
      playlist: "ranked",
      format: "duo",
    });
    expect(searching.phase).toBe("searching");
    expect(searching.ticketId).toBe("ticket-1");
    expect(searching.joinPending).toBe(false);

    const matched = reduceQueue(searching, {
      type: "matched",
      version: 2,
      matchId: "match-9",
    });
    expect(matched.phase).toBe("matched");
    expect(matched.matchId).toBe("match-9");
    expect(matched.status).toBe("matched");
  });

  it("ignores stale queue versions and gaps to repair", () => {
    const state = searchingQueue({ version: 5 });
    expect(
      reduceQueue(state, {
        type: "matched",
        version: 4,
        matchId: "old",
      }),
    ).toBe(state);
    expect(
      reduceQueue(state, {
        type: "matched",
        version: 5,
        matchId: "dup",
      }),
    ).toBe(state);

    const gap = reduceQueue(state, {
      type: "matched",
      version: 8,
      matchId: "skipped",
    });
    expect(gap.needsSnapshotRepair).toBe(true);
    expect(gap.phase).toBe("repairing");
    expect(gap.matchId).toBeNull();
  });

  it("pauses optimistic join while repairing", () => {
    const state = reduceQueue(createInitialQueueProjection(0), {
      type: "reconnect",
    });
    const optimistic = reduceQueue(state, {
      type: "join_optimistic",
      playlist: "casual",
      format: "solo",
    });
    expect(optimistic.joinPending).toBe(false);
    expect(optimistic).toBe(state);
  });

  it("accepts optimistic join when healthy and idle", () => {
    const idle = createInitialQueueProjection(0);
    const pending = reduceQueue(idle, {
      type: "join_optimistic",
      playlist: "casual",
      format: "squad",
    });
    expect(pending.joinPending).toBe(true);
    expect(pending.playlist).toBe("casual");
    expect(pending.format).toBe("squad");
  });

  it("leaves queue and repairs from snapshot", () => {
    const state = searchingQueue({ version: 3 });
    const left = reduceQueue(state, { type: "left", version: 4 });
    expect(left.phase).toBe("idle");
    expect(left.ticketId).toBeNull();
    expect(left.status).toBe("not_queued");

    const degraded = reduceQueue(state, {
      type: "degraded",
      reason: "ws_closed",
    });
    const fixed = applyQueueSnapshot(degraded, {
      ticketId: "ticket-2",
      status: "matched",
      playlist: "casual",
      format: "solo",
      matchId: "match-1",
      version: 9,
    });
    expect(fixed.phase).toBe("matched");
    expect(fixed.matchId).toBe("match-1");
    expect(fixed.version).toBe(9);
    expect(fixed.needsSnapshotRepair).toBe(false);
    expect(fixed.joinPending).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// Party projection
// ---------------------------------------------------------------------------

describe("reduceParty / applyPartySnapshot", () => {
  function hydratedParty(
    overrides: Partial<PartySnapshot> = {},
  ): PartyProjection {
    return applyPartySnapshot(createInitialPartyProjection("party-1"), {
      partyId: "party-1",
      status: "lobby",
      leaderId: "leader-1",
      memberCount: 2,
      readyCount: 1,
      version: 3,
      phase: "lobby",
      ...overrides,
    });
  }

  it("applies member and ready deltas", () => {
    const state = hydratedParty();
    const members = reduceParty(state, {
      type: "member_updated",
      version: 4,
      memberCount: 4,
      readyCount: 2,
    });
    expect(members.memberCount).toBe(4);
    expect(members.readyCount).toBe(2);

    const ready = reduceParty(members, {
      type: "ready_changed",
      version: 5,
      readyCount: 4,
    });
    expect(ready.readyCount).toBe(4);
  });

  it("ignores old/duplicate and gaps to repair", () => {
    const state = hydratedParty({ version: 6 });
    expect(
      reduceParty(state, {
        type: "queued",
        version: 5,
      }),
    ).toBe(state);
    expect(
      reduceParty(state, {
        type: "queued",
        version: 6,
      }),
    ).toBe(state);

    const gap = reduceParty(state, { type: "queued", version: 10 });
    expect(gap.needsSnapshotRepair).toBe(true);
    expect(gap.phase).toBe("repairing");
    expect(gap.status).toBe("lobby");
  });

  it("transitions queued -> matched -> terminal", () => {
    let state = hydratedParty({ version: 3 });
    state = reduceParty(state, { type: "queued", version: 4 });
    expect(state.phase).toBe("queued");
    state = reduceParty(state, { type: "matched", version: 5 });
    expect(state.phase).toBe("matched");
    state = reduceParty(state, {
      type: "terminal",
      version: 6,
      status: "disbanded",
    });
    expect(state.phase).toBe("terminal");
    expect(state.status).toBe("disbanded");
  });

  it("reconnect forces repair; snapshot clears it", () => {
    const state = hydratedParty();
    const reconnect = reduceParty(state, { type: "reconnect" });
    expect(reconnect.needsSnapshotRepair).toBe(true);
    expect(reconnect.optimisticPaused).toBe(true);

    const fixed = applyPartySnapshot(reconnect, {
      partyId: "party-1",
      status: "ready",
      leaderId: "leader-1",
      memberCount: 4,
      readyCount: 4,
      version: 12,
    });
    expect(fixed.phase).toBe("ready");
    expect(fixed.version).toBe(12);
    expect(fixed.needsSnapshotRepair).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// Match projection
// ---------------------------------------------------------------------------

describe("reduceMatch / applyMatchSnapshot", () => {
  function hydratedMatch(
    overrides: Partial<MatchSnapshot> = {},
  ): MatchProjection {
    return applyMatchSnapshot(createInitialMatchProjection("match-1"), {
      matchId: "match-1",
      status: "in_progress",
      currentRoundNumber: 1,
      teamScore: 0,
      opponentScore: 0,
      version: 2,
      phase: "playing",
      ...overrides,
    });
  }

  it("applies contiguous score and reveal events", () => {
    const state = hydratedMatch();
    const revealed = reduceMatch(state, {
      type: "reveal",
      version: 3,
      teamScore: 2400,
      opponentScore: 1800,
    });
    expect(revealed.phase).toBe("reveal");
    expect(revealed.teamScore).toBe(2400);
    expect(revealed.opponentScore).toBe(1800);

    const advanced = reduceMatch(revealed, {
      type: "round_advanced",
      version: 4,
      currentRoundNumber: 2,
    });
    expect(advanced.phase).toBe("playing");
    expect(advanced.currentRoundNumber).toBe(2);
    expect(advanced.guessPending).toBe(false);
  });

  it("ignores old/duplicate match versions", () => {
    const state = hydratedMatch({ version: 7, teamScore: 1000 });
    expect(
      reduceMatch(state, {
        type: "versioned_hint",
        version: 6,
        teamScore: 0,
      }),
    ).toBe(state);
    expect(
      reduceMatch(state, {
        type: "versioned_hint",
        version: 7,
        teamScore: 0,
      }),
    ).toBe(state);
  });

  it("gaps pause optimistic guess submission", () => {
    const state = hydratedMatch({ version: 1 });
    const gap = reduceMatch(state, {
      type: "versioned_hint",
      version: 4,
      teamScore: 9999,
    });
    expect(gap.needsSnapshotRepair).toBe(true);
    expect(gap.optimisticPaused).toBe(true);
    expect(gap.teamScore).toBe(0);

    const optimistic = reduceMatch(gap, {
      type: "guess_submitted_optimistic",
    });
    expect(optimistic.guessPending).toBe(false);
  });

  it("allows optimistic guess while healthy and playing", () => {
    const state = hydratedMatch();
    const pending = reduceMatch(state, {
      type: "guess_submitted_optimistic",
    });
    expect(pending.guessPending).toBe(true);
  });

  it("reconnect and degraded require snapshot repair", () => {
    const state = hydratedMatch();
    const reconnect = reduceMatch(state, { type: "reconnect" });
    expect(reconnect.phase).toBe("repairing");
    expect(reconnect.needsSnapshotRepair).toBe(true);

    const degraded = reduceMatch(state, {
      type: "degraded",
      reason: "heartbeat_timeout",
    });
    expect(degraded.lastError).toBe("heartbeat_timeout");
  });

  it("terminal event and authoritative snapshot recovery", () => {
    const state = hydratedMatch({ version: 5 });
    const terminal = reduceMatch(state, {
      type: "terminal",
      version: 6,
      status: "completed",
      teamScore: 5000,
      opponentScore: 4200,
    });
    expect(terminal.phase).toBe("terminal");
    expect(terminal.teamScore).toBe(5000);

    const broken = reduceMatch(state, { type: "gap_detected" });
    const fixed = applyMatchSnapshot(broken, {
      matchId: "match-1",
      status: "completed",
      currentRoundNumber: 5,
      teamScore: 5100,
      opponentScore: 4300,
      version: 15,
    });
    expect(fixed.version).toBe(15);
    expect(fixed.phase).toBe("terminal");
    expect(fixed.needsSnapshotRepair).toBe(false);
    expect(fixed.optimisticPaused).toBe(false);
    expect(fixed.guessPending).toBe(false);
  });
});
