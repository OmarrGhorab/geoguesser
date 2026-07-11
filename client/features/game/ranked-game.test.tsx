import { describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NextIntlClientProvider } from "next-intl";
import { RankedGame } from "./ranked-game";
import type { RankedGameDTO, RankedRoundDTO } from "./ranked-types";
import en from "@/messages/en.json";

vi.mock("./ranked-actions", () => ({
  submitRankedGuessAction: vi.fn(async () => ({
    ok: true,
    kind: "guess",
    result: {
      guess: {
        id: "guess-1",
        latitude: 10,
        longitude: 20,
        distance_meters: 1000,
        score: 4000,
        submitted_at: "2026-07-11T12:00:10Z",
      },
      actual_location: {
        latitude: 10.1,
        longitude: 20.1,
        country_code: "US",
        region: null,
        locality: null,
      },
    },
  })),
  reloadRankedRoundAction: vi.fn(async () => ({
    ok: true,
    kind: "reload",
    game: {
      id: "game-1",
      mode: "ranked",
      status: "active",
      map_id: "map-1",
      round_count: 5,
      timer_seconds: 60,
      scoring_version: 1,
      current_round_number: 1,
      total_score: 4000,
      started_at: "2020-01-01T00:00:00Z",
      completed_at: null,
    },
    round: {
      id: "round-1",
      round_number: 1,
      status: "active",
      starts_at: "2020-01-01T00:00:00Z",
      ends_at: "2099-01-01T00:00:00Z",
      media: null,
    },
  })),
  loadRankedResultsAction: vi.fn(),
}));

const baseGame: RankedGameDTO = {
  id: "game-1",
  mode: "ranked",
  status: "active",
  map_id: "map-1",
  round_count: 5,
  timer_seconds: 60,
  scoring_version: 1,
  current_round_number: 1,
  total_score: 0,
  started_at: "2026-07-11T12:00:00Z",
  completed_at: null,
};

const baseRound: RankedRoundDTO = {
  id: "round-1",
  round_number: 1,
  status: "active",
  starts_at: "2020-01-01T00:00:00Z",
  ends_at: "2099-01-01T00:00:00Z",
  media: null,
};

function renderGame(round: RankedRoundDTO | null = baseRound, game: RankedGameDTO = baseGame) {
  return render(
    <NextIntlClientProvider locale="en" messages={en}>
      <RankedGame gameId="game-1" initialGame={game} initialRound={round} />
    </NextIntlClientProvider>,
  );
}

describe("RankedGame", () => {
  it("disables early guess before scheduled start", async () => {
    const future = new Date(Date.now() + 60_000).toISOString();
    renderGame({ ...baseRound, starts_at: future });
    expect(await screen.findByText(/starts in/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /submit guess/i })).toBeDisabled();
  });

  it("allows one submission when round is live", async () => {
    const user = userEvent.setup();
    renderGame();
    const submit = screen.getByRole("button", { name: /submit guess/i });
    await waitFor(() => expect(submit).toBeEnabled());
    await user.click(submit);
    const { submitRankedGuessAction } = await import("./ranked-actions");
    await waitFor(() => expect(submitRankedGuessAction).toHaveBeenCalled());
    expect(await screen.findByText(/guess scored 4000 points/i)).toBeInTheDocument();
  });

  it("exposes accessible labels for guess controls", () => {
    renderGame();
    expect(screen.getByLabelText(/guess latitude/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/guess longitude/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /refresh round/i })).toBeInTheDocument();
  });

  it("renders results with polite status", () => {
    renderGame(null, {
      ...baseGame,
      status: "completed",
      completed_at: "2026-07-11T12:30:00Z",
    });
    expect(screen.getByRole("heading", { name: /ranked results/i })).toBeInTheDocument();
  });
});
