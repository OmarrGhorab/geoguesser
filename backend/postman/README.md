# GeoGuess Postman

## Import
1. Postman → **Import**
2. Select:
   - `GeoGuess-API.postman_collection.json`
   - `GeoGuess-Local.postman_environment.json`
3. Choose environment **GeoGuess Local**

## Quick start
1. Run **01 Auth → Get current session (bootstrap cookies)**
2. Set `csrfToken` if the test script did not (copy `csrf_token` cookie)
3. Run **Register** or **Login**
4. Explore folders for Health, Maps, Games, Rooms, Friends, etc.
5. Use **17 Multiplayer Modes & Chat** for Practice, Party Lobby, canonical Ranked queues, and team chat/moderation.

## Auth notes
- Cookie sessions (`access_token`, `refresh_token`, optional `guest_session`)
- Mutations need header `X-CSRF-Token: {{csrfToken}}` matching the cookie
- Keep Postman's cookie jar enabled

## Examples
- Example responses are embedded on each request (Examples dropdown in Postman).
- Raw mode fixtures are also available in `saved-responses/`, including Practice history/guess and Party Lobby defaults/standings.

## Multiplayer mode flows

- **Practice**: create a Practice session, submit a guess, create the next round after completion, inspect history, then end the session.
- **Party Lobby**: create the room with its 180-second default timer, have players join and mark ready, then start it as host.
- **Ranked**: use the canonical Ranked solo, duo, or squad queue requests; duo and squad require `partyId`.
- **Match chat**: team-only Duo/Squad chat supports message listing, text/attachment messages, mute/unmute, and reporting.

## Coverage

Domain folders in `GeoGuess-API.postman_collection.json`:

| Folder | Domain |
|--------|--------|
| 00 Health | Liveness, readiness, metrics |
| 01 Auth | Session bootstrap, register/login, OAuth |
| 02 Profile | Current profile, public stats, history |
| 18 Home | Authenticated home aggregate (`GET /home`) |
| 03 Uploads & Files | Presign, complete, signed URLs |
| 04 Maps | List/get maps |
| 05 Locations | Location media |
| 06 Games (Solo) | Create, Quick Play, rounds, guess, timeout, shared/final results |
| 07 Challenges | Daily and shared challenges |
| 08 Missions & Streaks | Missions and daily streak |
| 09 Rooms | Lobby create/join/settings/ready/start, kick, self-leave, host cancel |
| 10 Matchmaking | Queue join/leave/status |
| 11 Friends | Requests, list, block |
| 12 Leaderboards | Global, daily, map, friends |
| 13 Realtime | Room/match/party WebSocket references, ticket issue |
| 14 Parties (Casual) | Premade duo/squad parties and invites |
| 15 Matches (Casual) | Match snapshot, round/terminal results, leave |
| 16 Competitive (Ranks & Seasons) | Profile, leaderboard, history, seasons |
| 17 Multiplayer Modes & Chat | Practice, Party Lobby, Ranked queues, team chat |

Useful collection variables: `baseUrl`, `realtimeBaseUrl`, `csrfToken`, `gameId`, `roundId`, `roomCode`, `matchId`, `partyId`, `mapId`.

## Sources
- `backend/openapi/openapi.yaml`
- `backend/internal/app/routes.go`
