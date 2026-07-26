"use client";

/**
 * Room realtime: prefer WebSocket when the browser can reach the API origin
 * with session cookies; always fall back to HTTP snapshot polling (works with
 * ngrok frontend + local API).
 */

export type RoomRealtimeEvent = {
  type?: string;
  room_code?: string;
  version?: number;
  payload?: unknown;
};

export type RoomRealtimeHandle = {
  stop: () => void;
};

function realtimeBaseUrl(): string | null {
  const raw = process.env.NEXT_PUBLIC_REALTIME_URL?.trim();
  if (!raw) return null;
  return raw.replace(/\/$/, "");
}

/**
 * Subscribe to room updates. Calls onHint for WS events and onPollTick for
 * polling intervals so the UI can refetch the HTTP snapshot (source of truth).
 */
export function connectRoomRealtime(options: {
  roomCode: string;
  pollMs?: number;
  onHint?: (event: RoomRealtimeEvent) => void;
  onPollTick: () => void;
  onStatus?: (status: "connecting" | "live" | "polling" | "offline") => void;
}): RoomRealtimeHandle {
  const pollMs = options.pollMs ?? 2_000;
  let stopped = false;
  let ws: WebSocket | null = null;
  let pollTimer: ReturnType<typeof setInterval> | null = null;
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null;

  const startPolling = () => {
    if (stopped || pollTimer) return;
    options.onStatus?.("polling");
    options.onPollTick();
    pollTimer = setInterval(() => {
      if (!stopped) options.onPollTick();
    }, pollMs);
  };

  const stopPolling = () => {
    if (pollTimer) {
      clearInterval(pollTimer);
      pollTimer = null;
    }
  };

  const connectWs = () => {
    const base = realtimeBaseUrl();
    if (!base || stopped) {
      startPolling();
      return;
    }
    // Room sockets use cookie session, not tickets (see backend realtime.Room).
    // When frontend origin ≠ API origin (ngrok → localhost), cookies won't send;
    // onerror/onclose falls back to polling.
    options.onStatus?.("connecting");
    try {
      const url = `${base}/rooms/${encodeURIComponent(options.roomCode)}`;
      ws = new WebSocket(url);
    } catch {
      startPolling();
      return;
    }

    ws.onopen = () => {
      if (stopped) return;
      options.onStatus?.("live");
      // Keep a slow poll as repair even when WS is live.
      if (!pollTimer) {
        pollTimer = setInterval(() => {
          if (!stopped) options.onPollTick();
        }, Math.max(pollMs * 2, 4_000));
      }
    };

    ws.onmessage = (message) => {
      if (stopped) return;
      try {
        const data = JSON.parse(String(message.data)) as RoomRealtimeEvent;
        options.onHint?.(data);
        // Snapshot repair on every meaningful event.
        options.onPollTick();
      } catch {
        options.onPollTick();
      }
    };

    ws.onerror = () => {
      // Browser will also fire onclose.
    };

    ws.onclose = () => {
      ws = null;
      if (stopped) return;
      stopPolling();
      startPolling();
      reconnectTimer = setTimeout(() => {
        if (!stopped) connectWs();
      }, 5_000);
    };
  };

  connectWs();
  // Always begin with an immediate HTTP snapshot.
  options.onPollTick();

  return {
    stop: () => {
      stopped = true;
      stopPolling();
      if (reconnectTimer) clearTimeout(reconnectTimer);
      if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
        ws.close();
      }
      ws = null;
      options.onStatus?.("offline");
    },
  };
}
