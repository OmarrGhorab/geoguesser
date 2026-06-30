import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Lobby } from "./lobby";
import type { Room } from "./types";

describe("Lobby", () => {
  it("renders room code, roster, host, and presence", () => {
    const room: Room = {
      id: "room-1",
      code: "ABC123",
      visibility: "private",
      status: "lobby",
      game_id: "game-1",
      host_player_id: "player-1",
      current_player_id: "player-1",
      version: 1,
      max_players: 8,
      round_count: 5,
      timer_seconds: null,
      expires_at: "2026-06-30T12:00:00Z",
      ready_player_ids: [],
      players: [
        {
          id: "player-1",
          user_id: null,
          display_name: "Host",
          role: "host",
          membership_status: "joined",
          presence_status: "connected",
          is_ready: false,
          total_score: 0,
          joined_at: "2026-06-30T12:00:00Z",
          left_at: null,
        },
      ],
    };

    render(
      <Lobby
        room={room}
        labels={{
          roomCode: "Room code",
          copyInvite: "Copy invite",
          players: "Players",
          host: "Host",
          connected: "Connected",
          disconnected: "Disconnected",
          rounds: "Rounds",
          timer: "Timer",
          noTimer: "No timer",
          hostControls: {
            title: "Host controls",
            roundCount: "Rounds",
            maxPlayers: "Max players",
            timerSeconds: "Timer seconds",
            saveSettings: "Save settings",
            startRoom: "Start room",
            removePlayer: "Remove",
            ready: "Ready",
            notReady: "Not ready",
            locked: "Settings locked",
          },
        }}
      />,
    );

    expect(screen.getByRole("heading", { name: "ABC123" })).toBeInTheDocument();
    expect(screen.getAllByText("Host").length).toBeGreaterThan(0);
    expect(screen.getByLabelText("Connected")).toBeInTheDocument();
  });
});
