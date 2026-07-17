import { describe, expect, it } from "vitest";
import {
  COMPASS_CENTER_TICK,
  COMPASS_TICK_WIDTH_PX,
  compassTickLabel,
  compassTrackOffset,
  normalizedHeading,
} from "./compass";

describe("mission compass", () => {
  it("normalizes headings in both directions", () => {
    expect(normalizedHeading(-90)).toBe(270);
    expect(normalizedHeading(450)).toBe(90);
  });

  it("moves the strip left as the player turns right", () => {
    expect(compassTrackOffset(90)).toBeLessThan(compassTrackOffset(0));
    expect(compassTrackOffset(0)).toBe(
      -(
        COMPASS_CENTER_TICK * COMPASS_TICK_WIDTH_PX +
        COMPASS_TICK_WIDTH_PX / 2
      ),
    );
  });

  it("labels the cardinal and intercardinal ticks", () => {
    expect(compassTickLabel(COMPASS_CENTER_TICK)).toBe("N");
    expect(compassTickLabel(COMPASS_CENTER_TICK + 4)).toBe("NE");
    expect(compassTickLabel(COMPASS_CENTER_TICK + 8)).toBe("E");
    expect(compassTickLabel(COMPASS_CENTER_TICK + 1)).toBeNull();
  });
});
