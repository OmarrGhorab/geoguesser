import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NextIntlClientProvider } from "next-intl";
import { MatchmakingPanel } from "./matchmaking-panel";
import type { MatchmakingStatusResponse } from "./types";
import en from "@/messages/en.json";
import { joinMatchmakingAction, leaveMatchmakingAction } from "./actions";

const push = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push }),
}));

vi.mock("./actions", () => ({
  joinMatchmakingAction: vi.fn(),
  leaveMatchmakingAction: vi.fn(),
}));

function renderPanel(status: MatchmakingStatusResponse) {
  return render(
    <NextIntlClientProvider locale="en" messages={en}>
      <MatchmakingPanel locale="en" initialStatus={status} />
    </NextIntlClientProvider>,
  );
}

const searchingStatus: MatchmakingStatusResponse = {
  status: "searching",
  queue: {
    mode: "ranked_standard",
    search_started_at: "2026-07-11T12:00:00Z",
    lease_expires_at: "2026-07-11T12:00:30Z",
  },
  match: null,
};

describe("MatchmakingPanel", () => {
  beforeEach(() => {
    vi.resetAllMocks();
    vi.useRealTimers();
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it("renders not queued join control", () => {
    renderPanel({ status: "not_queued", queue: null, match: null });
    expect(screen.getByRole("button", { name: /find ranked match/i })).toBeInTheDocument();
  });

  it("renders searching state with leave control and non-color indicator", () => {
    renderPanel(searchingStatus);
    expect(screen.getByText(/searching for a ranked match/i)).toBeInTheDocument();
    expect(screen.getByText(/^searching$/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /leave queue/i })).toBeInTheDocument();
  });

  it("disables join while pending", async () => {
    const user = userEvent.setup();
    let resolveJoin: ((value: Awaited<ReturnType<typeof joinMatchmakingAction>>) => void) | undefined;
    vi.mocked(joinMatchmakingAction).mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveJoin = resolve;
        }),
    );
    renderPanel({ status: "not_queued", queue: null, match: null });
    const button = screen.getByRole("button", { name: /find ranked match/i });
    await user.click(button);
    await waitFor(() => expect(button).toBeDisabled());
    await act(async () => {
      resolveJoin?.({
        ok: true,
        status: searchingStatus,
      });
    });
  });

  it("polls while searching without overlapping requests and stops on matched", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          status: "matched",
          queue: null,
          match: {
            match_id: "m1",
            game_id: "g1",
            mode: "ranked_standard",
            formed_at: "2026-07-11T12:01:00Z",
            destination: "/games/g1",
          },
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    renderPanel(searchingStatus);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(2100);
    });
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(push).toHaveBeenCalledWith("/en/games/g1"));

    await act(async () => {
      await vi.advanceTimersByTimeAsync(4000);
    });
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("handles leave claim race by recovering via status poll", async () => {
    const user = userEvent.setup();
    vi.mocked(leaveMatchmakingAction).mockResolvedValue({
      ok: false,
      code: "matchmaking_claim_in_progress",
      message: "claim in progress",
    });
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          status: "matched",
          queue: null,
          match: {
            match_id: "m1",
            game_id: "g1",
            mode: "ranked_standard",
            formed_at: "2026-07-11T12:01:00Z",
            destination: "/games/g1",
          },
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    renderPanel(searchingStatus);

    await user.click(screen.getByRole("button", { name: /leave queue/i }));
    await waitFor(() => expect(leaveMatchmakingAction).toHaveBeenCalled());
    await waitFor(() => expect(fetchMock).toHaveBeenCalled());
    await waitFor(() => expect(push).toHaveBeenCalledWith("/en/games/g1"));
  });

  it("announces errors with alert role", async () => {
    const user = userEvent.setup();
    vi.mocked(joinMatchmakingAction).mockResolvedValue({
      ok: false,
      code: "matchmaking_account_ineligible",
      message: "Not eligible.",
    });
    renderPanel({ status: "not_queued", queue: null, match: null });
    await user.click(screen.getByRole("button", { name: /find ranked match/i }));
    expect(await screen.findByRole("alert")).toHaveTextContent(/not eligible/i);
  });
});
