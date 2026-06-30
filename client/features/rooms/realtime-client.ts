import type { RoomEventEnvelope } from "./realtime-types";

type RoomRealtimeClientOptions = {
  url: string;
  onEvent: (event: RoomEventEnvelope) => void;
  onStateChange?: (state: "connecting" | "connected" | "reconnecting" | "degraded" | "closed") => void;
};

export class RoomRealtimeClient {
  private socket: WebSocket | null = null;
  private closed = false;

  constructor(private readonly options: RoomRealtimeClientOptions) {}

  connect() {
    this.closed = false;
    this.options.onStateChange?.("connecting");
    this.socket = new WebSocket(this.options.url);
    this.socket.addEventListener("open", () => this.options.onStateChange?.("connected"));
    this.socket.addEventListener("message", (message) => {
      const event = parseRoomEvent(message.data);
      if (event) {
        this.options.onEvent(event);
      }
    });
    this.socket.addEventListener("close", () => {
      this.options.onStateChange?.(this.closed ? "closed" : "reconnecting");
    });
    this.socket.addEventListener("error", () => this.options.onStateChange?.("degraded"));
  }

  close() {
    this.closed = true;
    this.socket?.close();
    this.socket = null;
  }
}

export function parseRoomEvent(raw: unknown): RoomEventEnvelope | null {
  if (typeof raw !== "string") {
    return null;
  }
  try {
    const parsed = JSON.parse(raw) as Partial<RoomEventEnvelope>;
    if (!parsed.event_id || !parsed.type || !parsed.room_code || typeof parsed.version !== "number" || !parsed.occurred_at) {
      return null;
    }
    return parsed as RoomEventEnvelope;
  } catch {
    return null;
  }
}
