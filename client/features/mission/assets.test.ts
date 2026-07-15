import { readFileSync } from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";
import { MISSION_ASSET_PATHS, MISSION_ASSETS, isMissionAssetPath } from "./assets";

describe("mission public assets contract", () => {
  it("lists only /mission/* paths used by the design screen", () => {
    expect(MISSION_ASSETS.background).toBe("/mission/mission-play-bg.png");
    expect(MISSION_ASSETS.map).toBe("/mission/mission-map.png");
    expect(MISSION_ASSETS.trophy).toBe("/mission/tournement.png");
    expect(MISSION_ASSETS.fiveK).toBe("/mission/5k.png");
    expect(MISSION_ASSETS.players).toBe("/mission/players-today.png");
    expect(MISSION_ASSET_PATHS).toHaveLength(5);
    for (const p of MISSION_ASSET_PATHS) {
      expect(p.startsWith("/mission/")).toBe(true);
      expect(isMissionAssetPath(p)).toBe(true);
    }
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
    expect(source.toLowerCase()).not.toContain("data-testid=\"mission-a-badge\"");
    expect(source).toContain("MISSION_ASSETS.background");
    expect(source).toContain("MISSION_ASSETS.map");
    expect(source).toContain("MISSION_ASSETS.trophy");
    expect(source).toContain("MISSION_ASSETS.fiveK");
    expect(source).toContain("MISSION_ASSETS.players");
    expect(source).toContain("Does **not** render the design");
  });
});
