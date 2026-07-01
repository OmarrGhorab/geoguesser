import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithIntl } from "./test-utils";
import { PublicStats } from "./public-stats";
import type { PublicProfileDTO, StatsDTO } from "./types";

function profileFixture(overrides: Partial<PublicProfileDTO> = {}): PublicProfileDTO {
  return {
    user_id: "user-1",
    display_name: "Player One",
    avatar_url: null,
    country_code: "US",
    ...overrides,
  };
}

function statsFixture(overrides: Partial<StatsDTO> = {}): StatsDTO {
  return {
    games_played: 5,
    total_score: 1000,
    average_score: 200,
    best_score: 500,
    last_played_at: null,
    ...overrides,
  };
}

describe("PublicStats", () => {
  it("renders the display name, country code, and stats", () => {
    renderWithIntl(<PublicStats profile={profileFixture()} stats={statsFixture()} />);

    expect(screen.getByRole("heading", { name: "Player One" })).toBeInTheDocument();
    expect(screen.getByText("US")).toBeInTheDocument();
    expect(screen.getByText("5")).toBeInTheDocument();
    expect(screen.getByText("200.0")).toBeInTheDocument();
  });

  it("shows an empty state when no games have been played", () => {
    renderWithIntl(<PublicStats profile={profileFixture()} stats={statsFixture({ games_played: 0 })} />);

    expect(screen.getByText("No completed games yet.")).toBeInTheDocument();
  });

  it("omits the country code when absent", () => {
    renderWithIntl(<PublicStats profile={profileFixture({ country_code: null })} stats={statsFixture()} />);

    expect(screen.queryByText("US")).not.toBeInTheDocument();
  });
});
