import { readFileSync } from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";
import {
  MISSION_ASSET_PATHS,
  MISSION_ASSETS,
  isMissionAssetPath,
} from "./assets";

describe("mission public assets contract", () => {
  it("lists only /mission/* paths used by the design screen", () => {
    expect(MISSION_ASSETS.background).toBe("/mission/mission-play-bg.png");
    expect(MISSION_ASSETS.map).toBe("/mission/mission-map.png");
    expect(MISSION_ASSETS.trophy).toBe("/mission/tournement.png");
    expect(MISSION_ASSETS.fiveK).toBe("/mission/5k.png");
    expect(MISSION_ASSETS.players).toBe("/mission/players-today.png");
    expect(MISSION_ASSETS.afterMission).toBe("/mission/after-mission.png");
    expect(MISSION_ASSETS.prizeWorld).toBe("/mission/prize-world.png");
    expect(MISSION_ASSETS.summaryTrophy).toBe("/mission/trophy.png");
    expect(MISSION_ASSET_PATHS).toHaveLength(8);
    for (const p of MISSION_ASSET_PATHS) {
      expect(p.startsWith("/mission/")).toBe(true);
      expect(isMissionAssetPath(p)).toBe(true);
    }
  });

  it("completed mission screen uses the played-today artwork", () => {
    const screenPath = path.resolve(
      __dirname,
      "./components/played-today-mission-screen.tsx",
    );
    const source = readFileSync(screenPath, "utf8");
    const resultsSource = readFileSync(
      path.resolve(__dirname, "./components/daily-results-screen.tsx"),
      "utf8",
    );
    const routeMapSource = readFileSync(
      path.resolve(__dirname, "./components/mission-route-map.tsx"),
      "utf8",
    );

    expect(source).toContain("MISSION_ASSETS.afterMission");
    expect(source).toContain("MISSION_ASSETS.prizeWorld");
    expect(resultsSource).toContain("MISSION_ASSETS.summaryTrophy");
    expect(resultsSource).toContain("DailyResultsMap");
    expect(routeMapSource).toContain("MISSION_ASSETS.background");
    expect(source).toContain("MissionRouteMap");
    expect(source).toContain("DailyResultsPanel");
    expect(source).toContain('href="#daily-results-detail"');
    expect(source).toContain('id="daily-results-detail"');
    expect(source).toContain("played-today-mission-screen");
    const overviewScreen = source.indexOf(
      'data-testid="played-today-overview-screen"',
    );
    const resultsScreen = source.indexOf(
      'data-testid="played-today-results-screen"',
    );
    expect(overviewScreen).toBeGreaterThan(-1);
    expect(resultsScreen).toBeGreaterThan(overviewScreen);
  });

  it("ships every mission asset file under client/public/mission", () => {
    const publicRoot = path.resolve(__dirname, "../../public");
    for (const asset of MISSION_ASSET_PATHS) {
      const filePath = path.join(publicRoot, asset.replace(/^\//, ""));
      const buf = readFileSync(filePath);
      expect(buf.byteLength).toBeGreaterThan(1000);
    }
  });

  it("daily mission screen source references mission assets and omits A badge", () => {
    const screenPath = path.resolve(
      __dirname,
      "./components/daily-mission-screen.tsx",
    );
    const source = readFileSync(screenPath, "utf8");
    for (const asset of MISSION_ASSET_PATHS) {
      expect(source.includes(asset) || source.includes("MISSION_ASSETS")).toBe(
        true,
      );
    }
    // Must not render design A badge control (literal A badge UI)
    expect(source).not.toMatch(/>\s*A\s*</);
    expect(source.toLowerCase()).not.toContain('data-testid="mission-a-badge"');
    expect(source).toContain("MISSION_ASSETS.background");
    expect(source).toContain("MISSION_ASSETS.map");
    expect(source).toContain("MISSION_ASSETS.trophy");
    expect(source).toContain("MISSION_ASSETS.fiveK");
    expect(source).toContain("MISSION_ASSETS.players");
    expect(source).toContain("mission-players-strip");
    expect(source).toContain("Does **not** render the design");
  });

  it("renders a dedicated historical empty state without a playable form", () => {
    const screenPath = path.resolve(
      __dirname,
      "./components/past-daily-mission-empty-screen.tsx",
    );
    const source = readFileSync(screenPath, "utf8");

    expect(source).toContain('data-testid="past-daily-mission-empty-screen"');
    expect(source).toContain("MISSION_ASSETS.background");
    expect(source).toContain('const todayHref = `/${locale}/daily-mission`');
    expect(source).not.toContain("<form");
  });
});
