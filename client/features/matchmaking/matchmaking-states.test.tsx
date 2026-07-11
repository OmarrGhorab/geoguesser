import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { NextIntlClientProvider } from "next-intl";
import {
  MatchmakingLoadingSkeleton,
  MatchmakingStatePanel,
  summarizeStatus,
} from "./matchmaking-states";
import en from "@/messages/en.json";
import ar from "@/messages/ar.json";

describe("matchmaking-states", () => {
  it("renders state panel with alert role", () => {
    render(<MatchmakingStatePanel title="Unavailable" message="Try again" role="alert" />);
    expect(screen.getByRole("alert")).toHaveTextContent("Unavailable");
  });

  it("renders loading skeleton", () => {
    render(<MatchmakingLoadingSkeleton />);
    expect(screen.getByLabelText(/loading matchmaking/i)).toBeInTheDocument();
  });

  it("renders non-color status indicator text", () => {
    render(
      <MatchmakingStatePanel title="Queue status" message="Wait" indicator="In queue" />,
    );
    expect(screen.getByText("In queue")).toBeInTheDocument();
  });

  it("summarizes public statuses", () => {
    expect(summarizeStatus({ status: "not_queued", queue: null, match: null })).toBe("not_queued");
    expect(
      summarizeStatus({
        status: "searching",
        queue: {
          mode: "ranked_standard",
          search_started_at: "2026-07-11T12:00:00Z",
          lease_expires_at: "2026-07-11T12:00:30Z",
        },
        match: null,
      }),
    ).toBe("searching");
  });

  it("composes localized English and Arabic state copy with direction-safe markup", () => {
    const { rerender } = render(
      <NextIntlClientProvider locale="en" messages={en}>
        <div dir="ltr">
          <MatchmakingStatePanel
            title={en.Matchmaking.notQueued.title}
            message={en.Matchmaking.notQueued.message}
          />
        </div>
      </NextIntlClientProvider>,
    );
    expect(screen.getByRole("heading", { name: en.Matchmaking.notQueued.title })).toBeInTheDocument();

    rerender(
      <NextIntlClientProvider locale="ar" messages={ar}>
        <div dir="rtl">
          <MatchmakingStatePanel
            title={ar.Matchmaking.notQueued.title}
            message={ar.Matchmaking.notQueued.message}
          />
        </div>
      </NextIntlClientProvider>,
    );
    expect(screen.getByRole("heading", { name: ar.Matchmaking.notQueued.title })).toBeInTheDocument();
    expect(screen.getByText(ar.Matchmaking.notQueued.message).closest("div[dir='rtl']")).toBeTruthy();
  });
});
