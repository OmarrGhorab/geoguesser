import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { RoomGame } from "./room-game";
import type { Room } from "./types";

const labels = {
  activeTitle: "Room game",
  round: "Round",
  countdown: "Time left",
  untimed: "Untimed round",
  progress: "Guess progress",
  submitted: "submitted",
  results: "Round results",
  score: "Score",
  final: "Final results",
  waiting: "Waiting for the next round",
  recovery: {
    reconnecting: "Reconnecting to the room",
    degraded: "Live updates are delayed",
    disconnected: "Disconnected from live updates",
    restored: "Live room state restored",
  },
};

function roomFixture(status: Room["status"] = "active"): Room {
  return {
    id: "room-1",
    code: "ABC123",
    visibility: "private",
    status,
    game_id: "game-1",
    host_player_id: "player-1",
    current_player_id: "player-1",
    version: 2,
    max_players: 8,
    round_count: 5,
    timer_seconds: 60,
    expires_at: "2026-06-30T12:00:00Z",
    ready_player_ids: [],
    current_round:
      status === "active"
        ? {
            id: "round-1",
            round_number: 1,
            status: "active",
            starts_at: "2026-06-30T12:00:00Z",
            ends_at: null,
            media: null,
            revealed: false,
          }
        : undefined,
    guess_progress: status === "active" ? { submitted_count: 1, eligible_count: 2, submitted_player_ids: ["player-1"] } : undefined,
    players: [
      {
        id: "player-1",
        user_id: null,
        display_name: "Host",
        role: "host",
        membership_status: "joined",
        presence_status: "connected",
        is_ready: true,
        total_score: 5000,
        joined_at: "2026-06-30T12:00:00Z",
        left_at: null,
      },
    ],
  };
}

describe("RoomGame", () => {
  it("renders active round and aggregate guess progress", () => {
    render(<RoomGame room={roomFixture()} labels={labels} />);

    expect(screen.getByRole("heading", { name: "Room game" })).toBeInTheDocument();
    expect(screen.getByText("Round 1")).toBeInTheDocument();
    expect(screen.getByText("1/2")).toBeInTheDocument();
  });

  it("renders final scoreboard for completed rooms", () => {
    render(<RoomGame room={roomFixture("completed")} labels={labels} />);

    expect(screen.getByRole("heading", { name: "Final results" })).toBeInTheDocument();
    expect(screen.getByText("Score: 5000")).toBeInTheDocument();
  });
});
