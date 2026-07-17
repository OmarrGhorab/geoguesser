import { createServer } from "node:http";

const port = Number(process.env.E2E_BACKEND_PORT ?? 8081);
const now = new Date();
const today = now.toISOString().slice(0, 10);
const resetEndsAt = new Date(
  now.getTime() + 24 * 60 * 60 * 1_000,
).toISOString();

const dailyMission = {
  challenge: {
    id: "11111111-1111-4111-8111-111111111111",
    type: "daily",
    seed: "playwright-daily-challenge",
    challenge_date: today,
    reset_starts_at: now.toISOString(),
    reset_ends_at: resetEndsAt,
    map: { id: "22222222-2222-4222-8222-222222222222" },
    settings: {
      round_count: 5,
      timer_seconds: 180,
      movement_rules: "standard",
      scoring_version: 1,
    },
    status: "active",
  },
  attempt_state: null,
  last_completed_game_id: null,
  streak: {
    current_count: 0,
    best_count: 0,
    status: "inactive",
    protection_state: "none",
    guest_limited: false,
  },
  missions_summary: [],
  leaderboard_summary: { participants: 4 },
  countdown: {
    reset_ends_at: resetEndsAt,
    seconds_remaining: 86_400,
  },
  daily_games_played: 0,
  daily_games_total: 5,
};

function sendJSON(response, status, body) {
  response.writeHead(status, {
    "Content-Type": "application/json; charset=utf-8",
    "Cache-Control": "no-store",
  });
  response.end(JSON.stringify(body));
}

const server = createServer((request, response) => {
  const url = new URL(request.url ?? "/", `http://${request.headers.host}`);
  if (url.pathname === "/health") {
    sendJSON(response, 200, { status: "ok" });
    return;
  }
  if (url.pathname === "/api/v1/home") {
    sendJSON(response, 401, {
      error: { code: "unauthorized", message: "Authentication required." },
    });
    return;
  }
  if (url.pathname === "/api/v1/challenges/daily") {
    sendJSON(response, 200, dailyMission);
    return;
  }
  sendJSON(response, 404, {
    error: { code: "not_found", message: "Not found." },
  });
});

server.listen(port, "127.0.0.1");

function shutdown() {
  server.close(() => process.exit(0));
}

process.on("SIGINT", shutdown);
process.on("SIGTERM", shutdown);
