/**
 * One-shot maintenance script: adds missing GeoGuess API endpoints to the
 * Postman collection. Its output is already committed in
 * GeoGuess-API.postman_collection.json.
 *
 * Re-running is safe but normally unnecessary: every insertion is guarded by
 * hasRequest()/ensureVar(), so a re-run on an up-to-date collection is a
 * no-op (randomUUID only feeds the console report, never the file).
 *
 * The API contract's source of truth is backend/openapi/openapi.yaml — when
 * endpoints change, update the spec first and mirror the collection (either
 * by extending this script or editing the collection directly).
 *
 * Run: node backend/postman/add-missing-endpoints.mjs
 */
import { readFileSync, writeFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";
import { randomUUID } from "node:crypto";

const __dirname = dirname(fileURLToPath(import.meta.url));
const collectionPath = join(__dirname, "GeoGuess-API.postman_collection.json");

const collection = JSON.parse(readFileSync(collectionPath, "utf8"));

function countRequests(items) {
  let n = 0;
  for (const it of items) {
    if (it.item) n += countRequests(it.item);
    else n++;
  }
  return n;
}

function findFolder(items, name) {
  return items.find((it) => it.name === name && Array.isArray(it.item));
}

function ensureVar(key, value = "") {
  if (!collection.variable.some((v) => v.key === key)) {
    collection.variable.push({ key, value });
  }
}

function hasRequest(folder, name) {
  return folder.item.some((it) => it.name === name && it.request);
}

function baseUrlParts(pathSegments) {
  return {
    raw: `{{baseUrl}}/${pathSegments.join("/")}`,
    host: ["{{baseUrl}}"],
    path: pathSegments,
  };
}

function acceptHeader() {
  return { key: "Accept", value: "application/json", type: "text" };
}

function contentTypeHeader() {
  return { key: "Content-Type", value: "application/json", type: "text" };
}

function csrfHeader() {
  return { key: "X-CSRF-Token", value: "{{csrfToken}}", type: "text" };
}

function idempotencyHeader(optional = false) {
  return {
    key: "Idempotency-Key",
    value: "{{$guid}}",
    type: "text",
    ...(optional ? { disabled: true } : {}),
  };
}

function emptyResponseOk() {
  return [];
}

function makeRequest({ name, method, pathSegments, headers, description, body, response }) {
  const req = {
    name,
    request: {
      method,
      header: headers,
      url: baseUrlParts(pathSegments),
      description: description || "",
    },
    response: response ?? emptyResponseOk(),
  };
  if (body !== undefined) {
    req.request.body = {
      mode: "raw",
      raw: typeof body === "string" ? body : JSON.stringify(body, null, 2),
      options: { raw: { language: "json" } },
    };
  }
  return req;
}

function makeWsReference({ name, rawUrl, host, port, path, description }) {
  const url = {
    raw: rawUrl,
    host,
    path,
  };
  if (port) url.port = port;
  return {
    name,
    request: {
      method: "GET",
      header: [acceptHeader()],
      url,
      description,
    },
    response: [],
  };
}

const before = countRequests(collection.item);

// --- Variables ---
ensureVar("baseUrl", "http://localhost:8080/api/v1");
ensureVar("csrfToken", "");
ensureVar("gameId", "");
ensureVar("roundId", "");
ensureVar("roomCode", "");
ensureVar("matchId", "00000000-0000-4000-8000-0000000000c2");
ensureVar("partyId", "00000000-0000-4000-8000-0000000000b1");
ensureVar("mapId", "039106ee-88e5-4fdd-b2d5-61760f5833f5");
ensureVar("realtimeBaseUrl", "http://localhost:8080");

// --- 06 Games (Solo) ---
const games = findFolder(collection.item, "06 Games (Solo)");
if (!games) throw new Error("Games folder not found");

const gameAdds = [
  makeRequest({
    name: "Quick Play",
    method: "POST",
    pathSegments: ["games", "quick-play"],
    headers: [acceptHeader(), contentTypeHeader(), csrfHeader(), idempotencyHeader(false)],
    description:
      "Server-owned Quick Play (5 rounds, 60s timer, default map). Returns active game. Requires session cookies and CSRF. Idempotency-Key recommended for create.",
    body: {},
  }),
  makeRequest({
    name: "Expire round (timeout)",
    method: "POST",
    pathSegments: ["games", "{{gameId}}", "rounds", "{{roundId}}", "timeout"],
    headers: [acceptHeader(), csrfHeader(), idempotencyHeader(true)],
    description:
      "Server-authoritative round timeout (Daily / solo timed modes). Records a zero-score timeout after the server deadline and advances or completes the game. Optional Idempotency-Key.",
  }),
  makeRequest({
    name: "Get shared round results",
    method: "GET",
    pathSegments: ["games", "{{gameId}}", "rounds", "{{roundId}}", "results"],
    headers: [acceptHeader()],
    description:
      "Shared multiplayer / Party Lobby revealed round results (participant-only). Available after shared round closure; 409 round_not_revealed while the round is still active.",
  }),
];

for (const req of gameAdds) {
  if (!hasRequest(games, req.name)) {
    // Insert Quick Play after Create game; others near related endpoints.
    if (req.name === "Quick Play") {
      const createIdx = games.item.findIndex((i) => i.name === "Create game");
      games.item.splice(createIdx >= 0 ? createIdx + 1 : games.item.length, 0, req);
    } else if (req.name === "Expire round (timeout)") {
      const guessIdx = games.item.findIndex((i) => i.name === "Submit guess");
      games.item.splice(guessIdx >= 0 ? guessIdx + 1 : games.item.length, 0, req);
    } else if (req.name === "Get shared round results") {
      const resultsIdx = games.item.findIndex((i) => i.name === "Get game results");
      games.item.splice(resultsIdx >= 0 ? resultsIdx : games.item.length, 0, req);
    } else {
      games.item.push(req);
    }
  }
}

// --- 18 Home (new folder) ---
let home = findFolder(collection.item, "18 Home");
if (!home) {
  home = {
    name: "18 Home",
    description: "Authenticated home aggregate for registered sessions.",
    item: [],
  };
  // Place after Profile (02) for discoverability.
  const profileIdx = collection.item.findIndex((i) => i.name === "02 Profile");
  collection.item.splice(profileIdx >= 0 ? profileIdx + 1 : collection.item.length, 0, home);
}

if (!hasRequest(home, "Get authenticated home")) {
  home.item.push(
    makeRequest({
      name: "Get authenticated home",
      method: "GET",
      pathSegments: ["home"],
      headers: [acceptHeader()],
      description:
        "Authenticated home aggregate for a registered session. Returns viewer profile, completed-game stats, daily challenge metadata, and a small set of active public maps. Requires registered auth cookies (401 for guests).",
    })
  );
}

// --- 09 Rooms ---
const rooms = findFolder(collection.item, "09 Rooms");
if (!rooms) throw new Error("Rooms folder not found");

const roomAdds = [
  makeRequest({
    name: "Leave room (self)",
    method: "DELETE",
    pathSegments: ["rooms", "{{roomCode}}", "players", "me"],
    headers: [acceptHeader(), csrfHeader()],
    description:
      "Self-leave for a non-host member. Idempotent leave when already left. Lobby hosts must cancel the room instead (409 host_action_required).",
  }),
  makeRequest({
    name: "Cancel room (host)",
    method: "DELETE",
    pathSegments: ["rooms", "{{roomCode}}"],
    headers: [acceptHeader(), csrfHeader()],
    description:
      "Host cancels a lobby before the game starts. Idempotent once cancelled. Returns 409 after the hosted game has started.",
  }),
];

for (const req of roomAdds) {
  if (!hasRequest(rooms, req.name)) {
    rooms.item.push(req);
  }
}

// --- 13 Realtime ---
const realtime = findFolder(collection.item, "13 Realtime");
if (!realtime) throw new Error("Realtime folder not found");

// Optionally improve existing room WS to use realtimeBaseUrl if still hard-coded.
const roomWs = realtime.item.find((i) => i.name === "Room WebSocket URL (reference)");
if (roomWs?.request?.url?.raw?.includes("http://localhost:8080/realtime/rooms")) {
  roomWs.request.url = {
    raw: "{{realtimeBaseUrl}}/realtime/rooms/{{roomCode}}",
    host: ["{{realtimeBaseUrl}}"],
    path: ["realtime", "rooms", "{{roomCode}}"],
  };
  roomWs.request.description =
    "WebSocket endpoint (NOT under /api/v1).\n\nConnect with: `ws://localhost:8080/realtime/rooms/{{roomCode}}` (or swap scheme when using {{realtimeBaseUrl}}).\nSend session cookies from auth (Postman WebSocket or wscat).";
}

const realtimeAdds = [
  makeRequest({
    name: "Issue realtime ticket",
    method: "POST",
    pathSegments: ["realtime", "tickets"],
    headers: [acceptHeader(), contentTypeHeader(), csrfHeader()],
    description:
      "Issue a short-lived one-time WebSocket ticket for an authorized party or match channel. Registered session + CSRF required. Offer `geoguess.v1` and `ticket.{opaque}` via Sec-WebSocket-Protocol; tickets never appear in URLs.",
    body: {
      channel_kind: "match",
      channel_id: "{{matchId}}",
    },
  }),
  makeWsReference({
    name: "Match WebSocket URL (reference)",
    rawUrl: "{{realtimeBaseUrl}}/realtime/matches/{{matchId}}",
    host: ["{{realtimeBaseUrl}}"],
    path: ["realtime", "matches", "{{matchId}}"],
    description:
      "WebSocket endpoint (NOT under /api/v1).\n\nConnect with: `ws://localhost:8080/realtime/matches/{{matchId}}`.\nPrefer a one-time ticket from **Issue realtime ticket** (`channel_kind: match`). Offer `geoguess.v1` and `ticket.{opaque}` subprotocols; send session cookies when required.",
  }),
  makeWsReference({
    name: "Party WebSocket URL (reference)",
    rawUrl: "{{realtimeBaseUrl}}/realtime/parties/{{partyId}}",
    host: ["{{realtimeBaseUrl}}"],
    path: ["realtime", "parties", "{{partyId}}"],
    description:
      "WebSocket endpoint (NOT under /api/v1).\n\nConnect with: `ws://localhost:8080/realtime/parties/{{partyId}}`.\nPrefer a one-time ticket from **Issue realtime ticket** (`channel_kind: party`). Offer `geoguess.v1` and `ticket.{opaque}` subprotocols; send session cookies when required.",
  }),
];

for (const req of realtimeAdds) {
  if (!hasRequest(realtime, req.name)) {
    realtime.item.push(req);
  }
}

const after = countRequests(collection.item);
const addedNames = [];

function collectNames(items, prefix = "") {
  for (const it of items) {
    if (it.item) collectNames(it.item, `${prefix}${it.name}/`);
    else addedNames.push(`${prefix}${it.name}`);
  }
}

// Only report newly added by re-diffing known set — store expected new names:
const expectedNew = [
  "Quick Play",
  "Expire round (timeout)",
  "Get shared round results",
  "Get authenticated home",
  "Leave room (self)",
  "Cancel room (host)",
  "Issue realtime ticket",
  "Match WebSocket URL (reference)",
  "Party WebSocket URL (reference)",
];

// Validate JSON round-trip
const serialized = JSON.stringify(collection, null, 2) + "\n";
JSON.parse(serialized);

// Uniqueness of method+raw url among leaf requests
const seen = new Map();
const dups = [];
function walkUniq(items) {
  for (const it of items) {
    if (it.item) walkUniq(it.item);
    else {
      const key = `${it.request?.method} ${it.request?.url?.raw || ""}`;
      if (seen.has(key)) dups.push(key);
      else seen.set(key, it.name);
    }
  }
}
walkUniq(collection.item);

writeFileSync(collectionPath, serialized, "utf8");

console.log(
  JSON.stringify(
    {
      before,
      after,
      delta: after - before,
      expectedNew,
      present: expectedNew.filter((n) => {
        let found = false;
        function find(items) {
          for (const it of items) {
            if (it.item) find(it.item);
            else if (it.name === n) found = true;
          }
        }
        find(collection.item);
        return found;
      }),
      variables: collection.variable.map((v) => v.key),
      duplicateMethodUrls: dups,
      idSample: randomUUID(),
    },
    null,
    2
  )
);
