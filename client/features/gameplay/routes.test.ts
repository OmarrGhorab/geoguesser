import { describe, expect, it } from "vitest";
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

describe("gameplay routes", () => {
  const gameId = "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee";
  const matchId = "ffffffff-1111-4222-8333-444444444444";

  it("prefixes every builder with the active locale", () => {
    expect(playHref("en", "practice")).toBe("/en/play?mode=practice");
    expect(playHref("ar")).toBe("/ar/play");
    expect(gameHref("ar", gameId)).toBe(`/ar/games/${gameId}`);
    expect(gameResultsHref("en", gameId)).toBe(
      `/en/games/${gameId}/results`,
    );
    expect(dailyMissionHref("ar")).toBe("/ar/daily-mission");
    expect(dailyMissionPlayHref("en", gameId)).toBe(
      `/en/daily-mission/play/${gameId}`,
    );
    expect(dailyMissionResultsHref("ar", gameId)).toBe(
      `/ar/daily-mission/results/${gameId}`,
    );
    expect(roomsHref("en", "party_lobby")).toBe(
      "/en/rooms?mode=party_lobby",
    );
    expect(roomHref("ar", "ZZ99")).toBe("/ar/rooms/ZZ99");
    expect(matchmakingHref("en")).toBe("/en/matchmaking");
    expect(matchmakingHref("ar", "casual_duo")).toBe(
      "/ar/matchmaking?mode=casual_duo",
    );
    expect(partyHref("en", gameId)).toBe(`/en/parties/${gameId}`);
    expect(matchHref("ar", matchId)).toBe(`/ar/matches/${matchId}`);
    expect(matchResultsHref("en", matchId)).toBe(
      `/en/matches/${matchId}/results`,
    );
  });

  it("localizes only known relative backend destinations", () => {
    expect(localizeBackendDestination("en", `/games/${gameId}`)).toBe(
      `/en/games/${gameId}`,
    );
    expect(
      localizeBackendDestination("ar", `/games/${gameId}/results`),
    ).toBe(`/ar/games/${gameId}/results`);
    expect(localizeBackendDestination("en", `/matches/${matchId}`)).toBe(
      `/en/matches/${matchId}`,
    );
    expect(localizeBackendDestination("en", "/matchmaking")).toBe(
      "/en/matchmaking",
    );
    expect(localizeBackendDestination("en", "/rooms")).toBe("/en/rooms");
    expect(localizeBackendDestination("en", "/daily-mission")).toBe(
      "/en/daily-mission",
    );
  });

  it("never localizes absolute or unknown destinations", () => {
    expect(
      localizeBackendDestination("en", "https://attacker.test/phish"),
    ).toBe(null);
    expect(localizeBackendDestination("en", "http://attacker.test")).toBe(
      null,
    );
    expect(localizeBackendDestination("en", "//attacker.test/x")).toBe(null);
    expect(localizeBackendDestination("en", "games/missing-slash")).toBe(
      null,
    );
    expect(localizeBackendDestination("en", "/admin")).toBe(null);
    expect(localizeBackendDestination("en", "/games/../secret")).toBe(null);
  });
});
