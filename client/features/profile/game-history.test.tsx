import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithIntl } from "./test-utils";
import { GameHistory } from "./game-history";
import type { GameHistoryItemDTO } from "./types";

function gameFixture(overrides: Partial<GameHistoryItemDTO> = {}): GameHistoryItemDTO {
  return {
    id: "game-1",
    map_id: "map-1",
    mode: "classic",
    status: "completed",
    round_count: 5,
    current_round_number: null,
    total_score: 3200,
    started_at: "2026-06-01T00:00:00Z",
    completed_at: "2026-06-01T00:10:00Z",
    created_at: "2026-06-01T00:00:00Z",
    ...overrides,
  };
}

describe("GameHistory", () => {
  it("shows an empty state when there are no games", () => {
    renderWithIntl(<GameHistory games={[]} page={{ limit: 10, next_cursor: null }} basePath="/en/profile" />);

    expect(screen.getByText("No games yet.")).toBeInTheDocument();
  });

  it("renders game entries with score and status", () => {
    renderWithIntl(<GameHistory games={[gameFixture()]} page={{ limit: 10, next_cursor: null }} basePath="/en/profile" />);

    expect(screen.getByText("Completed")).toBeInTheDocument();
    expect(screen.getByText("Score: 3200")).toBeInTheDocument();
    expect(screen.getByText("classic")).toBeInTheDocument();
  });

  it("shows round progress for active games", () => {
    renderWithIntl(
      <GameHistory
        games={[gameFixture({ status: "active", current_round_number: 3 })]}
        page={{ limit: 10, next_cursor: null }}
        basePath="/en/profile"
      />,
    );

    expect(screen.getByText("Round 3 of 5")).toBeInTheDocument();
  });

  it("renders a load more link when a next cursor is present", () => {
    renderWithIntl(<GameHistory games={[gameFixture()]} page={{ limit: 10, next_cursor: "abc123" }} basePath="/en/profile" />);

    const link = screen.getByRole("link", { name: "Load more" });
    expect(link).toHaveAttribute("href", "/en/profile?cursor=abc123");
  });

  it("does not render hidden answer or coordinate details", () => {
    renderWithIntl(<GameHistory games={[gameFixture()]} page={{ limit: 10, next_cursor: null }} basePath="/en/profile" />);

    expect(screen.queryByText(/latitude/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/longitude/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/provider/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/answer/i)).not.toBeInTheDocument();
  });

  it("renders Arabic empty history copy", () => {
    renderWithIntl(<GameHistory games={[]} page={{ limit: 10, next_cursor: null }} basePath="/ar/profile" />, "ar");

    expect(screen.getByText("لا توجد ألعاب بعد.")).toBeInTheDocument();
  });
});
