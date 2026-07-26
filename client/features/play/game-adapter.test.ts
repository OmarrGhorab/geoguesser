import { describe, expect, it } from "vitest";
import {
  getGameCapabilities,
  resolveSoloCapabilities,
} from "./game-adapter";

describe("getGameCapabilities", () => {
  it("marks solo as fixed-round with progression and immediate reveal", () => {
    const caps = getGameCapabilities("solo");
    expect(caps.mode).toBe("solo");
    expect(caps.fixedRounds).toBe(true);
    expect(caps.showProgression).toBe(true);
    expect(caps.immediateReveal).toBe(true);
    expect(caps.sharedReveal).toBe(false);
    expect(caps.openEnded).toBe(false);
    expect(caps.serverDefaults).toBe(false);
  });

  it("marks quick play as timed server-default solo-compatible", () => {
    const caps = getGameCapabilities("quick_play");
    expect(caps.timed).toBe(true);
    expect(caps.allowTimeout).toBe(true);
    expect(caps.fixedRounds).toBe(true);
    expect(caps.serverDefaults).toBe(true);
    expect(caps.openEnded).toBe(false);
  });

  it("marks practice as untimed open-ended with history and explicit end", () => {
    const caps = getGameCapabilities("practice");
    expect(caps.timed).toBe(false);
    expect(caps.allowTimeout).toBe(false);
    expect(caps.openEnded).toBe(true);
    expect(caps.fixedRounds).toBe(false);
    expect(caps.showProgression).toBe(false);
    expect(caps.history).toBe(true);
    expect(caps.explicitEnd).toBe(true);
    expect(caps.immediateReveal).toBe(true);
  });

  it("marks daily as timed fixed-round with timeout support", () => {
    const caps = getGameCapabilities("daily");
    expect(caps.timed).toBe(true);
    expect(caps.allowTimeout).toBe(true);
    expect(caps.fixedRounds).toBe(true);
    expect(caps.showProgression).toBe(true);
    expect(caps.serverDefaults).toBe(true);
  });
});

describe("resolveSoloCapabilities", () => {
  it("disables timeout when timer is null", () => {
    const caps = resolveSoloCapabilities({ timerSeconds: null });
    expect(caps.timed).toBe(false);
    expect(caps.allowTimeout).toBe(false);
  });

  it("enables timeout when timer is positive", () => {
    const caps = resolveSoloCapabilities({ timerSeconds: 90 });
    expect(caps.timed).toBe(true);
    expect(caps.allowTimeout).toBe(true);
  });
});
