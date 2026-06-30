import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { HostControls } from "./host-controls";
import type { Room } from "./types";

const labels = {
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
};

function roomFixture(): Room {
  return {
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
        is_ready: true,
        total_score: 0,
        joined_at: "2026-06-30T12:00:00Z",
        left_at: null,
      },
      {
        id: "player-2",
        user_id: null,
        display_name: "Guest",
        role: "player",
        membership_status: "joined",
        presence_status: "connected",
        is_ready: false,
        total_score: 0,
        joined_at: "2026-06-30T12:00:01Z",
        left_at: null,
      },
    ],
  };
}

describe("HostControls", () => {
  it("enables host commands and removal for the room host", () => {
    render(<HostControls room={roomFixture()} currentPlayerId="player-1" labels={labels} />);

    expect(screen.getByRole("button", { name: "Save settings" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Start room" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Remove" })).toBeEnabled();
  });

  it("keeps privileged controls disabled for non-host players", () => {
    render(<HostControls room={roomFixture()} currentPlayerId="player-2" labels={labels} />);

    expect(screen.getByRole("button", { name: "Save settings" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Start room" })).toBeDisabled();
    expect(screen.queryByRole("button", { name: "Remove" })).not.toBeInTheDocument();
  });
});
