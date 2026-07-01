import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithIntl } from "./test-utils";
import { ProfileForm } from "./profile-form";
import type { ProfileDTO } from "./types";

function profileFixture(overrides: Partial<ProfileDTO> = {}): ProfileDTO {
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
    ...overrides,
  };
}

describe("ProfileForm", () => {
  it("renders each editable field pre-filled with the current profile", () => {
    renderWithIntl(<ProfileForm profile={profileFixture()} />);

    expect(screen.getByDisplayValue("Player One")).toBeInTheDocument();
    expect(screen.getByDisplayValue("US")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save changes" })).toBeEnabled();
  });

  it("selects the profile's current locale option", () => {
    renderWithIntl(<ProfileForm profile={profileFixture({ locale: "ar" })} />);

    expect(screen.getByRole("option", { name: "Arabic", selected: true })).toBeInTheDocument();
  });
});
