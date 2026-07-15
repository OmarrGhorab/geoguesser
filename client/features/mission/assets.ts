/** Public assets for the daily-mission screen (`client/public/mission/`). */
export const MISSION_ASSETS = {
  background: "/mission/mission-play-bg.png",
  map: "/mission/mission-map.png",
  trophy: "/mission/tournement.png",
  fiveK: "/mission/5k.png",
  players: "/mission/players-today.png",
} as const;

export const MISSION_ASSET_PATHS = Object.values(MISSION_ASSETS);

export function isMissionAssetPath(path: string): boolean {
  return MISSION_ASSET_PATHS.some(
    (asset) => path === asset || path.includes(asset),
  );
}
