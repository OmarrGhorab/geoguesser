import { describe, expect, it } from "vitest";

import {
  gameplayErrorMessagePath,
  mapApiErrorToGameplay,
  mapApiErrorToGameplayCategory,
  recoveryActionForCategory,
} from "./errors";
import {
  createGameRequestSchema,
  currentRoundSchema,
  gameModeSchema,
  gameResultsSchema,
  gameSchema,
  gameStatusSchema,
  guessResultSchema,
  mapListResponseSchema,
  matchResultsSchema,
  matchSnapshotSchema,
  matchmakingStatusResponseSchema,
  partySchema,
  practiceHistorySchema,
  queueStatusSchema,
  quickPlayResponseSchema,
  realtimeTicketRequestSchema,
  realtimeTicketResponseSchema,
  roomSnapshotSchema,
  submitGuessBodySchema,
} from "./schemas";

const MAP_ID = "22222222-2222-4222-8222-222222222222";
const GAME_ID = "11111111-1111-4111-8111-111111111111";
const ROUND_ID = "33333333-3333-4333-8333-333333333333";
const GUESS_ID = "44444444-4444-4444-8444-444444444444";
const USER_ID = "55555555-5555-4555-8555-555555555555";
const PLAYER_ID = "66666666-6666-4666-8666-666666666666";
const MATCH_ID = "77777777-7777-4777-8777-777777777777";
const PARTY_ID = "88888888-8888-4888-8888-888888888888";

const baseGame = {
  id: GAME_ID,
  mode: "solo" as const,
  status: "active" as const,
  map_id: MAP_ID,
  round_count: 5,
  timer_seconds: 60,
  scoring_version: 1,
  current_round_number: 1,
  total_score: 0,
  started_at: "2026-07-19T12:00:00Z",
  completed_at: null,
};

describe("game schemas — success payloads", () => {
  it("parses a solo game and game response / quick play response", () => {
    const game = gameSchema.parse(baseGame);
    expect(game.mode).toBe("solo");
    expect(game.timer_seconds).toBe(60);

    const quick = quickPlayResponseSchema.parse({
      game: {
        ...baseGame,
        mode: "quick_play",
        status: "active",
        round_count: 5,
        timer_seconds: 60,
        current_round_number: 1,
      },
    });
    expect(quick.game.mode).toBe("quick_play");
  });

  it("accepts create game requests for solo and practice only", () => {
    expect(
      createGameRequestSchema.parse({
        mode: "solo",
        map_id: MAP_ID,
        round_count: 5,
        timer_seconds: 90,
      }),
    ).toMatchObject({ mode: "solo", map_id: MAP_ID });

    expect(
      createGameRequestSchema.parse({
        mode: "practice",
        map_id: MAP_ID,
        timer_seconds: null,
      }).mode,
    ).toBe("practice");

    expect(
      createGameRequestSchema.safeParse({
        mode: "quick_play",
        map_id: MAP_ID,
      }).success,
    ).toBe(false);

    expect(
      createGameRequestSchema.safeParse({
        mode: "daily",
        map_id: MAP_ID,
      }).success,
    ).toBe(false);
  });

  it("parses a reveal-safe current round without true coordinates", () => {
    const response = currentRoundSchema.parse({
      round: {
        id: ROUND_ID,
        round_number: 2,
        status: "active",
        starts_at: "2026-07-19T12:01:00Z",
        ends_at: null,
        media: {
          type: "panorama",
          panorama_id: "safe_Pano-ABCDEF12",
          attribution: null,
        },
      },
    });

    expect(response.round.media.panorama_id).toBe("safe_Pano-ABCDEF12");
    expect(response.round).not.toHaveProperty("latitude");
    expect(response.round).not.toHaveProperty("longitude");
    expect(JSON.stringify(response)).not.toMatch(/"latitude"/);
  });

  it("parses a solo guess result with actual location", () => {
    const result = guessResultSchema.parse({
      guess: {
        id: GUESS_ID,
        latitude: 30.0,
        longitude: 31.0,
        distance_meters: 1200,
        score: 4200,
        submitted_at: "2026-07-19T12:02:00Z",
        timed_out: false,
      },
      actual_location: {
        latitude: 30.04,
        longitude: 31.23,
        country_code: "EG",
        region: null,
        locality: "Cairo",
      },
      max_score: 5000,
      score_percent: 84,
      outcome: "close",
      round_completed: true,
      game_completed: false,
    });

    expect(result.actual_location?.country_code).toBe("EG");
    expect(result.outcome).toBe("close");
  });

  it("parses submit guess body bounds", () => {
    expect(
      submitGuessBodySchema.parse({ latitude: 0, longitude: 0 }),
    ).toEqual({ latitude: 0, longitude: 0 });
    expect(
      submitGuessBodySchema.safeParse({ latitude: 100, longitude: 0 }).success,
    ).toBe(false);
  });

  it("parses game results with flexible round counts (not hard-coded to 5)", () => {
    const threeRounds = gameResultsSchema.parse({
      game: { ...baseGame, status: "completed", total_score: 9000, mode: "solo" },
      players: [
        {
          id: PLAYER_ID,
          user_id: USER_ID,
          display_name: "Explorer",
          role: "host",
          status: "active",
          total_score: 9000,
        },
      ],
      rounds: [1, 2, 3].map((n) => ({
        round_id: `${n}3333333-3333-4333-8333-333333333333`,
        round_number: n,
        actual_location: {
          latitude: 10 + n,
          longitude: 20 + n,
          country_code: "EG",
        },
        guesses: [
          {
            id: `${n}4444444-4444-4444-8444-444444444444`,
            latitude: 11 + n,
            longitude: 21 + n,
            distance_meters: 500,
            score: 3000,
            submitted_at: "2026-07-19T12:00:00Z",
            timed_out: false,
          },
        ],
      })),
    });
    expect(threeRounds.rounds).toHaveLength(3);

    const emptyRounds = gameResultsSchema.parse({
      game: { ...baseGame, status: "completed", mode: "practice" },
      players: [],
      rounds: [],
    });
    expect(emptyRounds.rounds).toHaveLength(0);
  });

  it("parses practice history with cursor paging fields", () => {
    const history = practiceHistorySchema.parse({
      items: [
        {
          round: {
            id: ROUND_ID,
            round_number: 1,
            status: "completed",
            media: { type: "image", url: "https://cdn.example.com/r1.jpg" },
          },
          guess: {
            id: GUESS_ID,
            latitude: 1,
            longitude: 2,
            distance_meters: 100,
            score: 4800,
            submitted_at: "2026-07-19T12:00:00Z",
            timed_out: false,
          },
          actual_location: {
            latitude: 1.1,
            longitude: 2.1,
            country_code: "US",
          },
        },
      ],
      next_cursor: "cursor-abc",
      has_more: true,
    });
    expect(history.has_more).toBe(true);
    expect(history.next_cursor).toBe("cursor-abc");
  });

  it("parses map list responses", () => {
    const list = mapListResponseSchema.parse({
      data: [
        {
          id: MAP_ID,
          slug: "world",
          name: "World",
          description: null,
          visibility: "public",
          access_tier: "free",
          difficulty: "mixed",
          status: "active",
          created_at: "2026-01-01T00:00:00Z",
          updated_at: "2026-01-01T00:00:00Z",
        },
      ],
      page: { limit: 20, next_cursor: null },
    });
    expect(list.data[0]?.slug).toBe("world");
  });
});

describe("nullable fields", () => {
  it("accepts null timer, timestamps, and multiplayer actual_location", () => {
    const game = gameSchema.parse({
      ...baseGame,
      timer_seconds: null,
      started_at: null,
      completed_at: null,
      current_round_number: null,
      open_ended: true,
      mode: "practice",
    });
    expect(game.timer_seconds).toBeNull();
    expect(game.current_round_number).toBeNull();

    const multiplayerGuess = guessResultSchema.parse({
      guess: {
        id: GUESS_ID,
        latitude: 0,
        longitude: 0,
        distance_meters: 0,
        score: 0,
        submitted_at: "2026-07-19T12:00:00Z",
        timed_out: false,
      },
      actual_location: null,
      outcome: "submitted",
      round_completed: false,
    });
    expect(multiplayerGuess.actual_location).toBeNull();
  });

  it("accepts null queue and match slots on matchmaking status", () => {
    const idle = matchmakingStatusResponseSchema.parse({
      status: "not_queued",
      queue: null,
      match: null,
    });
    expect(idle.queue).toBeNull();
    expect(idle.match).toBeNull();
  });

  it("accepts null media and optional room fields", () => {
    const room = roomSnapshotSchema.parse({
      id: "99999999-9999-4999-8999-999999999999",
      game_id: null,
      host_player_id: PLAYER_ID,
      current_player_id: PLAYER_ID,
      code: "ABC123",
      visibility: "private",
      status: "lobby",
      version: 1,
      max_players: 50,
      round_count: 5,
      timer_seconds: null,
      expires_at: "2026-07-19T13:00:00Z",
      players: [
        {
          id: PLAYER_ID,
          user_id: USER_ID,
          display_name: "Host",
          role: "host",
          membership_status: "joined",
          presence_status: "connected",
          is_ready: true,
          total_score: 0,
          joined_at: "2026-07-19T12:00:00Z",
          left_at: null,
        },
      ],
      ready_player_ids: [PLAYER_ID],
      current_round: null,
      guess_progress: null,
      mode: "party_lobby",
    });
    expect(room.current_round).toBeNull();
    expect(room.timer_seconds).toBeNull();
  });
});

describe("stable status enums", () => {
  it("accepts every GameStatus value", () => {
    for (const status of [
      "pending",
      "active",
      "completed",
      "abandoned",
      "cancelled",
    ] as const) {
      expect(gameStatusSchema.parse(status)).toBe(status);
      expect(gameSchema.parse({ ...baseGame, status }).status).toBe(status);
    }
  });

  it("accepts every queue status", () => {
    for (const status of [
      "not_queued",
      "searching",
      "matched",
      "temporarily_unavailable",
    ] as const) {
      expect(queueStatusSchema.parse(status)).toBe(status);
    }
  });

  it("parses party, match snapshot, and terminal match results", () => {
    const party = partySchema.parse({
      id: PARTY_ID,
      format: "duo",
      capacity: 2,
      status: "forming",
      version: 3,
      leader_user_id: USER_ID,
      active_match_id: null,
      members: [
        {
          user_id: USER_ID,
          display_name: "Leader",
          avatar_url: null,
          ready: true,
          is_leader: true,
          joined_at: "2026-07-19T12:00:00Z",
        },
      ],
      created_at: "2026-07-19T12:00:00Z",
      updated_at: "2026-07-19T12:00:00Z",
    });
    expect(party.version).toBe(3);

    const snapshot = matchSnapshotSchema.parse({
      id: MATCH_ID,
      game_id: GAME_ID,
      playlist: "casual",
      format: "solo",
      status: "active",
      result: null,
      team_size: 1,
      viewer: {
        user_id: USER_ID,
        game_player_id: PLAYER_ID,
        team_slot: 1,
        submitted: false,
        can_chat: false,
        allowed_spectate_player_ids: [],
      },
      teams: [
        {
          slot: 1,
          score: 0,
          players: [
            {
              game_player_id: PLAYER_ID,
              user_id: USER_ID,
              display_name: "You",
              status: "active",
              submitted: false,
            },
          ],
        },
        {
          slot: 2,
          score: 0,
          players: [
            {
              game_player_id: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
              user_id: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb",
              display_name: "Opponent",
              status: "active",
              submitted: false,
            },
          ],
        },
      ],
      round: {
        id: ROUND_ID,
        number: 1,
        status: "active",
        starts_at: "2026-07-19T12:00:00Z",
        ends_at: null,
        media: { type: "panorama", panorama_id: "safe_match_pano1" },
        submitted_count: 0,
        eligible_count: 2,
      },
      last_round_result: null,
      team_markers: [],
      realtime_version: 1,
      formed_at: "2026-07-19T12:00:00Z",
      timer_seconds: null,
    });
    expect(snapshot.round?.media?.panorama_id).toBe("safe_match_pano1");
    expect(JSON.stringify(snapshot.round)).not.toMatch(/"latitude"/);

    const terminal = matchResultsSchema.parse({
      result: "team_one_win",
      winner_team_slot: 1,
      teams: snapshot.teams,
      rounds: [],
      progression: { applied: false, reason: "casual" },
    });
    expect(terminal.progression.reason).toBe("casual");
  });

  it("parses realtime ticket request/response", () => {
    expect(
      realtimeTicketRequestSchema.parse({
        channel_kind: "match",
        channel_id: MATCH_ID,
      }),
    ).toEqual({ channel_kind: "match", channel_id: MATCH_ID });

    const ticket = realtimeTicketResponseSchema.parse({
      ticket: "one-time-secret",
      expires_at: "2026-07-19T12:05:00Z",
      websocket_url: "/realtime/matches/" + MATCH_ID,
    });
    expect(ticket.ticket).toBe("one-time-secret");
  });
});

describe("legacy mode recovery on game.mode", () => {
  it("accepts every GameMode value including legacy aliases", () => {
    const modes = [
      "solo",
      "private_room",
      "party_lobby",
      "practice",
      "quick_play",
      "daily",
      "ranked",
      "casual_solo",
      "casual_duo",
      "casual_squad",
      "ranked_solo",
      "ranked_duo",
      "ranked_squad",
      "ranked_standard",
    ] as const;

    for (const mode of modes) {
      expect(gameModeSchema.parse(mode)).toBe(mode);
      const parsed = gameSchema.parse({ ...baseGame, mode });
      expect(parsed.mode).toBe(mode);
    }
  });

  it("rejects unknown modes on recovery reads", () => {
    expect(gameModeSchema.safeParse("unknown_mode").success).toBe(false);
    expect(
      gameSchema.safeParse({ ...baseGame, mode: "turbo_blitz" }).success,
    ).toBe(false);
  });
});

describe("safe error mapping", () => {
  it("maps explicit codes to stable categories", () => {
    expect(mapApiErrorToGameplayCategory(409, "idempotency_conflict")).toBe(
      "idempotency_conflict",
    );
    expect(mapApiErrorToGameplayCategory(409, "host_action_required")).toBe(
      "host_action_required",
    );
    expect(mapApiErrorToGameplayCategory(409, "wrong_game_mode")).toBe(
      "wrong_mode",
    );
    expect(mapApiErrorToGameplayCategory(202, "progression_pending")).toBe(
      "progression_pending",
    );
    expect(mapApiErrorToGameplayCategory(409, "round_not_revealed")).toBe(
      "round_not_revealed",
    );
    expect(mapApiErrorToGameplayCategory(404, "not_found")).toBe("not_found");
    expect(mapApiErrorToGameplayCategory(429, "rate_limited")).toBe(
      "rate_limited",
    );
    expect(mapApiErrorToGameplayCategory(503, "temporarily_unavailable")).toBe(
      "unavailable",
    );
  });

  it("falls back to status when code is missing or unknown", () => {
    expect(mapApiErrorToGameplayCategory(401)).toBe("unauthorized");
    expect(mapApiErrorToGameplayCategory(403)).toBe("forbidden");
    expect(mapApiErrorToGameplayCategory(404)).toBe("not_found");
    expect(mapApiErrorToGameplayCategory(409)).toBe("conflict");
    expect(mapApiErrorToGameplayCategory(422)).toBe("validation");
    expect(mapApiErrorToGameplayCategory(429)).toBe("rate_limited");
    expect(mapApiErrorToGameplayCategory(503)).toBe("unavailable");
    expect(mapApiErrorToGameplayCategory(500, "something_novel")).toBe(
      "internal",
    );
  });

  it("assigns recovery actions and message keys without leaking internals", () => {
    const mapped = mapApiErrorToGameplay(409, "idempotency_conflict");
    expect(mapped.category).toBe("idempotency_conflict");
    expect(mapped.recovery).toBe("refresh");
    expect(mapped.messageKey).toBe("idempotency_conflict");
    expect(mapped.messagePath).toBe("gameplay.errors.idempotency_conflict");
    expect(gameplayErrorMessagePath("unauthorized")).toBe(
      "gameplay.errors.unauthorized",
    );

    expect(recoveryActionForCategory("unauthorized")).toBe("login");
    expect(recoveryActionForCategory("rate_limited")).toBe("wait");
    expect(recoveryActionForCategory("progression_pending")).toBe("wait");
    expect(recoveryActionForCategory("host_action_required")).toBe("none");
    expect(recoveryActionForCategory("unavailable")).toBe("retry");
    expect(recoveryActionForCategory("not_found")).toBe("exit");
  });

  it("never surfaces raw backend stack or secret material in mapping helpers", () => {
    const leakyCode = "internal_error; secret=abc; stack=at foo";
    const category = mapApiErrorToGameplayCategory(500, leakyCode);
    // Unknown codes fall back to status — not echo the raw string as category.
    expect(category).toBe("internal");
    expect(category).not.toContain("secret");
    expect(gameplayErrorMessagePath(category)).toBe("gameplay.errors.internal");
  });
});
