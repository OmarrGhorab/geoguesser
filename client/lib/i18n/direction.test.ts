import { describe, expect, it } from "vitest";
import { getDirection } from "./direction";

describe("getDirection", () => {
  it("uses left-to-right direction for English", () => {
    expect(getDirection("en")).toBe("ltr");
  });

  it("uses right-to-left direction for Arabic", () => {
    expect(getDirection("ar")).toBe("rtl");
  });
});
