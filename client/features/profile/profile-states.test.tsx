import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithIntl } from "./test-utils";

vi.mock("@/lib/i18n/navigation", () => ({
  Link: ({ href, children, ...rest }: { href: string; children: React.ReactNode }) => (
    <a href={href} {...rest}>
      {children}
    </a>
  ),
}));

const {
  ProfileLoadingSkeleton,
  ProfileUnauthorizedPanel,
  ProfileNotFoundPanel,
  ProfileUnexpectedErrorPanel,
} = await import("./profile-states");

describe("profile-states", () => {
  it("renders an accessible loading skeleton", () => {
    renderWithIntl(<ProfileLoadingSkeleton />);

    expect(screen.getByRole("status")).toBeInTheDocument();
  });

  it("renders the unauthorized panel with a back-to-play link", () => {
    renderWithIntl(<ProfileUnauthorizedPanel />);

    expect(screen.getByText("Sign in required")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Back to play" })).toHaveAttribute("href", "/");
  });

  it("renders the not-found panel", () => {
    renderWithIntl(<ProfileNotFoundPanel />);

    expect(screen.getByText("Profile not found")).toBeInTheDocument();
  });

  it("renders the unexpected error panel with an alert role", () => {
    renderWithIntl(<ProfileUnexpectedErrorPanel />);

    expect(screen.getByRole("alert")).toBeInTheDocument();
    expect(screen.getByText("Something went wrong")).toBeInTheDocument();
  });
});
