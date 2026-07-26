import { describe, expect, it } from "vitest";
import {
  CANONICAL_MODES,
  LEGACY_ALIASES,
  getModeDefinition,
  isCanonicalMode,
  listSelectableModes,
  normalizeMode,
  type CanonicalMode,
} from "./modes";
import {
  dailyMissionHref,
  dailyMissionPlayHref,
  dailyMissionResultsHref,
  gameHref,
  gameResultsHref,
  localizeBackendDestination,
  matchHref,
  matchResultsHref,
  matchmakingHref,
  partyHref,
  playHref,
  roomHref,
  roomsHref,
} from "./routes";

describe("canonical mode registry", () => {
  it("exposes exactly eleven canonical modes in product order", () => {
    expect(CANONICAL_MODES).toEqual([
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
    ]);
    expect(CANONICAL_MODES).toHaveLength(11);
    expect(new Set(CANONICAL_MODES).size).toBe(11);
  });

  it("lists exactly eleven selectable modes without legacy aliases", () => {
    const selectable = listSelectableModes();
    expect(selectable).toHaveLength(11);

    const modes = selectable.map((entry) => entry.mode);
    expect(modes).toEqual([...CANONICAL_MODES]);
    expect(modes).not.toContain("private_room");
    expect(modes).not.toContain("ranked_standard");
    expect(modes).not.toContain("ranked");

    for (const alias of Object.keys(LEGACY_ALIASES)) {
      expect(modes).not.toContain(alias);
    }
  });

  it("recognizes only the eleven canonical values", () => {
    for (const mode of CANONICAL_MODES) {
      expect(isCanonicalMode(mode)).toBe(true);
    }
    expect(isCanonicalMode("private_room")).toBe(false);
    expect(isCanonicalMode("ranked")).toBe(false);
    expect(isCanonicalMode("ranked_standard")).toBe(false);
    expect(isCanonicalMode("")).toBe(false);
    expect(isCanonicalMode(null)).toBe(false);
  });

  it("normalizes legacy aliases one-way at recovery", () => {
    expect(normalizeMode("private_room")).toBe("party_lobby");
    expect(normalizeMode("ranked_standard")).toBe("ranked_solo");
    expect(normalizeMode("ranked")).toBe("ranked_solo");
    expect(normalizeMode("solo")).toBe("solo");
    expect(normalizeMode("practice")).toBe("practice");
    expect(normalizeMode("unknown_mode")).toBeNull();
    expect(normalizeMode("")).toBeNull();
    expect(normalizeMode(undefined)).toBeNull();
  });

  it("attaches session requirements, team sizes, and message keys", () => {
    const practice = getModeDefinition("practice");
    expect(practice.sessionRequirement).toBe("guest_or_account");
    expect(practice.teamSize).toBe(1);
    expect(practice.messageKey).toBe("Play.modes.practice");
    expect(practice.descriptionKey).toBe("Play.modes.practiceDescription");

    const rankedDuo = getModeDefinition("ranked_duo");
    expect(rankedDuo.sessionRequirement).toBe("account");
    expect(rankedDuo.teamSize).toBe(2);
    expect(rankedDuo.messageKey).toBe("Play.modes.ranked_duo");

    const casualSquad = getModeDefinition("casual_squad");
    expect(casualSquad.teamSize).toBe(4);
    expect(casualSquad.sessionRequirement).toBe("account");

    for (const mode of CANONICAL_MODES) {
      const def = getModeDefinition(mode);
      expect(def.messageKey).toBe(`Play.modes.${mode}`);
    }
  });

  it("differentiates practice vs ranked capabilities", () => {
    const practice = getModeDefinition("practice").capabilities;
    const rankedSolo = getModeDefinition("ranked_solo").capabilities;

    expect(practice.timed).toBe(false);
    expect(practice.openEnded).toBe(true);
    expect(practice.sharedReveal).toBe(false);
    expect(practice.speedBonus).toBe(false);
    expect(practice.progression).toBe(false);
    expect(practice.hostControls).toBe(false);

    expect(rankedSolo.timed).toBe(true);
    expect(rankedSolo.openEnded).toBe(false);
    expect(rankedSolo.sharedReveal).toBe(true);
    expect(rankedSolo.speedBonus).toBe(true);
    expect(rankedSolo.progression).toBe(true);
    expect(rankedSolo.teams).toBe(false);

    const party = getModeDefinition("party_lobby").capabilities;
    expect(party.hostControls).toBe(true);
    expect(party.sharedReveal).toBe(true);
    expect(party.progression).toBe(false);

    const rankedDuo = getModeDefinition("ranked_duo").capabilities;
    expect(rankedDuo.teams).toBe(true);
    expect(rankedDuo.chat).toBe(true);
    expect(rankedDuo.speedBonus).toBe(true);

    const casualSolo = getModeDefinition("casual_solo").capabilities;
    expect(casualSolo.timed).toBe(false);
    expect(casualSolo.speedBonus).toBe(false);
    expect(casualSolo.progression).toBe(false);
  });

  it("marks guest-eligible vs account-only modes consistently", () => {
    const guestOrAccount: CanonicalMode[] = [
      "solo",
      "practice",
      "daily",
      "quick_play",
      "party_lobby",
    ];
    const accountOnly: CanonicalMode[] = [
      "casual_solo",
      "casual_duo",
      "casual_squad",
      "ranked_solo",
      "ranked_duo",
      "ranked_squad",
    ];

    for (const mode of guestOrAccount) {
      expect(getModeDefinition(mode).sessionRequirement).toBe(
        "guest_or_account",
      );
    }
    for (const mode of accountOnly) {
      expect(getModeDefinition(mode).sessionRequirement).toBe("account");
    }
  });
});

describe("localized gameplay routes", () => {
  const gameId = "11111111-1111-4111-8111-111111111111";
  const matchId = "22222222-2222-4222-8222-222222222222";
  const partyId = "33333333-3333-4333-8333-333333333333";
  const roomCode = "ABCD12";

  it("preserves locale on play and game route builders", () => {
    expect(playHref("en")).toBe("/en/play");
    expect(playHref("ar", "solo")).toBe("/ar/play?mode=solo");
    expect(playHref("en", "quick_play")).toBe("/en/play?mode=quick_play");
    expect(gameHref("en", gameId)).toBe(`/en/games/${gameId}`);
    expect(gameResultsHref("ar", gameId)).toBe(
      `/ar/games/${gameId}/results`,
    );
  });

  it("builds daily mission, room, matchmaking, party, and match routes", () => {
    expect(dailyMissionHref("en")).toBe("/en/daily-mission");
    expect(dailyMissionPlayHref("en", gameId)).toBe(
      `/en/daily-mission/play/${gameId}`,
    );
    expect(dailyMissionResultsHref("ar", gameId)).toBe(
      `/ar/daily-mission/results/${gameId}`,
    );
    expect(roomsHref("en")).toBe("/en/rooms");
    expect(roomsHref("ar", "party_lobby")).toBe(
      "/ar/rooms?mode=party_lobby",
    );
    expect(roomHref("en", roomCode)).toBe(`/en/rooms/${roomCode}`);
    expect(matchmakingHref("en", "ranked_solo")).toBe(
      "/en/matchmaking?mode=ranked_solo",
    );
    expect(partyHref("ar", partyId)).toBe(`/ar/parties/${partyId}`);
    expect(matchHref("en", matchId)).toBe(`/en/matches/${matchId}`);
    expect(matchResultsHref("ar", matchId)).toBe(
      `/ar/matches/${matchId}/results`,
    );
  });

  it("localizes known backend destinations and rejects open redirects", () => {
    expect(localizeBackendDestination("en", `/games/${gameId}`)).toBe(
      `/en/games/${gameId}`,
    );
    expect(localizeBackendDestination("ar", `/matches/${matchId}`)).toBe(
      `/ar/matches/${matchId}`,
    );
    expect(
      localizeBackendDestination("en", `/matches/${matchId}/results`),
    ).toBe(`/en/matches/${matchId}/results`);
    expect(localizeBackendDestination("en", `/rooms/${roomCode}`)).toBe(
      `/en/rooms/${roomCode}`,
    );
    expect(localizeBackendDestination("ar", "/play")).toBe("/ar/play");
    expect(localizeBackendDestination("en", "/play?mode=solo")).toBe(
      "/en/play?mode=solo",
    );

    // Open redirect / unsafe
    expect(localizeBackendDestination("en", "https://evil.example/steal")).toBe(
      null,
    );
    expect(localizeBackendDestination("en", "//evil.example/steal")).toBe(
      null,
    );
    expect(localizeBackendDestination("en", "/\\evil.example")).toBe(null);
    expect(localizeBackendDestination("en", "/unknown/path")).toBe(null);
    expect(localizeBackendDestination("en", "")).toBe(null);
    expect(
      localizeBackendDestination("en", `/games/not-a-uuid`),
    ).toBe(null);
  });
});
