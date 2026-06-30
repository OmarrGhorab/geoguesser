import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { RecoveryBanner } from "./recovery";

const labels = {
  reconnecting: "Reconnecting to the room",
  degraded: "Live updates are delayed",
  disconnected: "Disconnected from live updates",
  restored: "Live room state restored",
};

describe("RecoveryBanner", () => {
  it("announces reconnecting and degraded states accessibly", () => {
    render(<RecoveryBanner state="reconnecting" labels={labels} />);
    expect(screen.getByRole("status")).toHaveTextContent("Reconnecting to the room");

    render(<RecoveryBanner state="degraded" labels={labels} />);
    expect(screen.getByRole("alert")).toHaveTextContent("Live updates are delayed");
  });
});
