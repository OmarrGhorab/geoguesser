import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithIntl } from "./test-utils";
import { ProfileForm } from "./profile-form";
import type { ProfileDTO } from "./types";

vi.mock("./actions", async () => {
  const actual = await vi.importActual<typeof import("./actions")>("./actions");
  return {
    ...actual,
    updateProfileAction: vi.fn(),
  };
});

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

  it("wires validation messages to accessible field descriptions", () => {
    renderWithIntl(<ProfileForm profile={profileFixture()} />);

    const displayName = screen.getByLabelText("Display name");
    expect(displayName).toHaveAttribute("minlength", "2");
    expect(displayName).toHaveAttribute("maxlength", "32");
    expect(screen.getByRole("status")).toHaveAccessibleName("");
  });

  it("does not render private auth or session fields", () => {
    renderWithIntl(<ProfileForm profile={profileFixture()} />);

    expect(screen.queryByText(/token/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/session/i)).not.toBeInTheDocument();
    expect(screen.queryByText("player@example.com")).not.toBeInTheDocument();
  });

  it("renders Arabic field labels through the same accessible controls", () => {
    renderWithIntl(<ProfileForm profile={profileFixture({ locale: "ar" })} />, "ar");

    expect(screen.getByLabelText("الاسم المعروض")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "حفظ التغييرات" })).toBeEnabled();
  });
});
