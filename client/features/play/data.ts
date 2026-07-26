import "server-only";

import { apiJson } from "@/lib/api/client";
import {
  currentRoundSchema,
  gameResponseSchema,
  gameResultsSchema,
  mapListResponseSchema,
  practiceHistorySchema,
  type CurrentRound,
  type Game,
  type GameResults,
  type PlayableMap,
  type PracticeHistory,
} from "@/features/play/schemas";

/** Prefer the seeded World map and deprioritize ephemeral integration-test maps. */
export function rankPlayableMaps(maps: readonly PlayableMap[]): PlayableMap[] {
  const rank = (map: PlayableMap): number => {
    const name = map.name.toLowerCase();
    const slug = map.slug.toLowerCase();
    if (slug === "world" || name === "world") return 0;
    if (
      name.includes("solo game") ||
      name.includes("test map") ||
      name.includes("team formation") ||
      slug.startsWith("solo-game-")
    ) {
      return 2;
    }
    return 1;
  };
  return [...maps].sort(
    (a, b) => rank(a) - rank(b) || a.name.localeCompare(b.name),
  );
}

export async function getPlayableMaps(options?: {
  limit?: number;
  cursor?: string;
}): Promise<{ maps: PlayableMap[]; nextCursor: string | null }> {
  const params = new URLSearchParams();
  if (options?.limit != null) params.set("limit", String(options.limit));
  if (options?.cursor) params.set("cursor", options.cursor);
  const query = params.toString();
  const response = await apiJson(
    `/maps${query ? `?${query}` : ""}`,
    mapListResponseSchema,
  );
  return {
    maps: rankPlayableMaps(response.data),
    nextCursor: response.page.next_cursor ?? null,
  };
}

/**
 * Loads public maps, paging until the preferred World map is found or pages
 * are exhausted, so setup does not default to 1-location test maps.
 */
export async function getPlaySetupMaps(
  maxPages = 4,
): Promise<{ maps: PlayableMap[] }> {
  const collected: PlayableMap[] = [];
  let cursor: string | undefined;
  for (let page = 0; page < maxPages; page += 1) {
    const result = await getPlayableMaps({ limit: 50, cursor });
    collected.push(...result.maps);
    const hasWorld = collected.some(
      (map) => map.slug === "world" || map.name.toLowerCase() === "world",
    );
    if (hasWorld || !result.nextCursor) {
      break;
    }
    cursor = result.nextCursor;
  }
  // Dedupe by id after multi-page fetch, then re-rank.
  const byId = new Map(collected.map((map) => [map.id, map]));
  return { maps: rankPlayableMaps([...byId.values()]) };
}

export async function getGame(gameId: string): Promise<Game> {
  const { game } = await apiJson(
    `/games/${gameId}`,
    gameResponseSchema,
    { requiresAuth: true },
  );
  return game;
}

export async function getCurrentRound(gameId: string): Promise<CurrentRound> {
  const { round } = await apiJson(
    `/games/${gameId}/rounds/current`,
    currentRoundSchema,
    { requiresAuth: true },
  );
  return round;
}

export async function getGameResults(gameId: string): Promise<GameResults> {
  return apiJson(`/games/${gameId}/results`, gameResultsSchema, {
    requiresAuth: true,
  });
}

export async function getPracticeHistory(
  gameId: string,
  cursor?: string,
  limit?: number,
): Promise<PracticeHistory> {
  const params = new URLSearchParams();
  if (cursor) params.set("cursor", cursor);
  if (limit != null) params.set("limit", String(limit));
  const query = params.toString();
  return apiJson(
    `/games/${gameId}/rounds${query ? `?${query}` : ""}`,
    practiceHistorySchema,
    { requiresAuth: true },
  );
}
