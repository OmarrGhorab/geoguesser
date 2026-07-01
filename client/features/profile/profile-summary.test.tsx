import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithIntl } from "./test-utils";
import { ProfileSummary } from "./profile-summary";
import type { ProfileDTO, StatsDTO } from "./types";

function profileFixture(): ProfileDTO {
  return {
    user_id: "user-1",
    email: "player@example.com",
    display_name: "Player One",
    avatar_url: null,
    country_code: "US",
    locale: "en",
    timezone: null,
    preferences: {},
    created_at: "2026-06-01T00:00:00Z",
    updated_at: "2026-06-01T00:00:00Z",
  };
}

function statsFixture(): StatsDTO {
  return {
    games_played: 10,
    total_score: 4200,
    average_score: 420.5,
    best_score: 990,
    last_played_at: "2026-06-30T00:00:00Z",
  };
}

describe("ProfileSummary", () => {
  it("renders the profile email and stats", () => {
    renderWithIntl(<ProfileSummary profile={profileFixture()} stats={statsFixture()} />);

    expect(screen.getByText("player@example.com")).toBeInTheDocument();
    expect(screen.getByText("10")).toBeInTheDocument();
    expect(screen.getByText("4200")).toBeInTheDocument();
    expect(screen.getByText("420.5")).toBeInTheDocument();
    expect(screen.getByText("990")).toBeInTheDocument();
  });

  it("renders the edit form with the current display name", () => {
    renderWithIntl(<ProfileSummary profile={profileFixture()} stats={statsFixture()} />);

    expect(screen.getByDisplayValue("Player One")).toBeInTheDocument();
  });
});
