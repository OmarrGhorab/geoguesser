import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { RoomStateMessage } from "./room-states";

const labels = {
  invalid: "Invalid room code",
  expired: "Room expired",
  full: "Room is full",
  kicked: "You were removed",
  unauthorized: "Not authorized",
  loading: "Loading room",
  empty: "No players yet",
  disabled: "Action unavailable",
  success: "Room ready",
  unexpected: "Something went wrong",
};

describe("RoomStateMessage", () => {
  it("uses alert semantics for failures", () => {
    render(<RoomStateMessage kind="invalid" labels={labels} />);
    expect(screen.getByRole("alert")).toHaveTextContent("Invalid room code");
  });

  it("uses status semantics for loading and success", () => {
    render(<RoomStateMessage kind="success" labels={labels} />);
    expect(screen.getByRole("status")).toHaveTextContent("Room ready");
  });
});
