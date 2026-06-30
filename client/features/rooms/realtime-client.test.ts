import { describe, expect, it } from "vitest";
import { parseRoomEvent } from "./realtime-client";

describe("parseRoomEvent", () => {
  it("accepts valid room event envelopes", () => {
    const event = parseRoomEvent(
      JSON.stringify({
        event_id: "evt_1",
        type: "room.snapshot",
        room_code: "ABC123",
        occurred_at: "2026-06-30T12:00:00Z",
        version: 1,
        payload: {},
      }),
    );

    expect(event?.type).toBe("room.snapshot");
  });

  it("rejects malformed event envelopes", () => {
    expect(parseRoomEvent("{")).toBeNull();
    expect(parseRoomEvent(JSON.stringify({ type: "room.snapshot" }))).toBeNull();
  });
});
