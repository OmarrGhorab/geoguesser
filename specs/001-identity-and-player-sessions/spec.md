# Feature Specification: Identity and Player Sessions

**Feature Branch**: `001-identity-and-player-sessions`

**Created**: 2026-06-25

**Status**: Approved

**Input**: User description: "start implementing phase 2 identity and player sessions. Make sure everything related to backend done. Need Google and Discord authentication beside normal email."

## User Scenarios & Testing

### User Story 1 - Register and Login with Email (Priority: P1)

A player can create a registered account using an email address and password, then log in to receive secure session cookies. The account unlocks persistent profile and registered-only features.

**Why this priority**: Email/password identity is the baseline authentication path and supports every other registered-user feature.

**Independent Test**: A new player can register, receive a success response with cookies, call `/auth/me`, log out, log back in, and call `/auth/me` again.

**Acceptance Scenarios**:

1. **Given** a valid email, password, and display name, **When** the player registers, **Then** a registered account is created and authentication cookies are set.
2. **Given** an existing account, **When** the player logs in with correct credentials, **Then** new rotated session cookies are set and the current session endpoint returns the registered user.
3. **Given** a logged-in player, **When** the player logs out, **Then** the refresh session is revoked and authentication cookies are cleared.

---

### User Story 2 - Google and Discord OAuth (Priority: P1)

A player can sign in or sign up using a Google or Discord account. If the provider identity matches an existing linked account, the player is logged in. If it is new, a registered account is created and linked to the provider identity.

**Why this priority**: The user explicitly requested Google and Discord authentication alongside email.

**Independent Test**: A player can initiate OAuth with Google or Discord, complete the provider flow, and return to an authenticated session with a registered account.

**Acceptance Scenarios**:

1. **Given** a player with no existing account, **When** they complete Google OAuth, **Then** a new registered account is created, linked to the Google identity, and session cookies are set.
2. **Given** a player whose registered account is already linked to Discord, **When** they complete Discord OAuth, **Then** they are logged in to that account and session cookies are set.
3. **Given** a logged-in registered player, **When** they complete OAuth with a new provider that is not yet linked, **Then** the provider identity is linked to the existing account.

---

### User Story 3 - Token Refresh and Session Rotation (Priority: P1)

A logged-in player can refresh their access token using a rotated refresh token. Old refresh tokens become invalid after rotation.

**Why this priority**: Secure cookie sessions depend on short-lived access tokens and rotated refresh tokens.

**Independent Test**: A player can log in, wait for the access token to expire, call refresh, receive a new access token, and continue using `/auth/me`.

**Acceptance Scenarios**:

1. **Given** a valid refresh token cookie, **When** the player calls refresh, **Then** a new access token and a new rotated refresh token are issued and the old refresh token is revoked.
2. **Given** a revoked or expired refresh token, **When** the player calls refresh, **Then** the request is rejected and cookies are cleared.

---

### User Story 4 - Guest Identity for Gameplay (Priority: P2)

A player who has not registered can receive a signed guest session that is sufficient to play solo games and join rooms.

**Why this priority**: Guest play lowers the barrier to entry and is required by the phase goals.

**Independent Test**: A new browser session can call `/auth/me` and receive a guest identity, then use that guest identity to create a solo game or join a room.

**Acceptance Scenarios**:

1. **Given** a browser with no auth or guest cookie, **When** the player visits gameplay endpoints, **Then** a signed guest session cookie is issued and `/auth/me` returns a guest session summary.
2. **Given** a guest session, **When** the player tries to access registered-only endpoints such as profile update, **Then** the request is rejected.

---

### User Story 5 - CSRF and Rate Limit Protection (Priority: P1)

Unsafe cookie-authenticated requests such as register, login, OAuth completion, refresh, and logout require a valid CSRF token. Auth-sensitive endpoints are rate limited to prevent abuse.

**Why this priority**: Required by the phase security model and protects accounts and OAuth flows from abuse.

**Independent Test**: A request to an unsafe auth endpoint without a valid CSRF token is rejected, and repeated failed login attempts are rate limited.

**Acceptance Scenarios**:

1. **Given** a valid login request without a CSRF token, **When** it is submitted, **Then** the request is rejected with a clear error.
2. **Given** repeated failed login attempts from the same actor, **When** the rate limit threshold is exceeded, **Then** subsequent attempts are rejected until the window resets.

---

### Edge Cases

- What happens when a player registers with an email that already exists?
- What happens when login credentials are incorrect?
- What happens when a refresh token is reused after rotation?
- What happens when a provider OAuth token exchange fails or is cancelled?
- What happens when a guest session cookie is tampered with or has an invalid signature?
- What loading, empty, error, disabled, and success states are visible to users on auth screens?
- How does the experience behave in English and Arabic, including RTL layout for auth forms?
- What performance or latency boundary would make the auth flow feel broken? Login/refresh should complete within a few hundred milliseconds.

## Requirements

### Functional Requirements

- **FR-001**: The system MUST allow players to register a registered account with email, password, and display name.
- **FR-002**: The system MUST allow players to log in with email and password.
- **FR-003**: The system MUST support sign-in and sign-up via Google OAuth.
- **FR-004**: The system MUST support sign-in and sign-up via Discord OAuth.
- **FR-005**: The system MUST allow a registered player to log out and revoke the current refresh session.
- **FR-006**: The system MUST rotate refresh tokens on every refresh request and revoke the previous token.
- **FR-007**: The system MUST issue short-lived access tokens and longer-lived refresh tokens as HTTP-only, Secure, SameSite cookies.
- **FR-008**: The system MUST issue and validate CSRF tokens for unsafe cookie-authenticated requests.
- **FR-009**: The system MUST support signed guest session cookies for anonymous gameplay.
- **FR-010**: The system MUST resolve the current session as anonymous, guest, or registered user via `/auth/me`.
- **FR-011**: The system MUST rate limit register, login, refresh, and OAuth callback endpoints.
- **FR-012**: The system MUST store refresh tokens only as hashes.
- **FR-013**: User-facing copy MUST be available in supported locale message catalogs for any visible auth text.
- **FR-014**: Interactive auth controls MUST expose accessible names and keyboard focus behavior.
- **FR-015**: API contract changes for auth MUST be reflected in the OpenAPI specification.

### Key Entities

- **Registered Account**: A player account with email, password hash, role, status, and timestamps. A registered account has one profile and many sessions.
- **User Profile**: Public identity fields for a registered account including display name, avatar, country, locale, and timezone.
- **Auth Session**: A refresh-token session owned by a registered account, storing a refresh token hash, user agent hash, IP address, expiration, and revocation state.
- **OAuth Connection**: A link between a registered account and an external provider identity such as Google or Discord, storing provider, provider account ID, and email snapshot.
- **Guest Session**: A signed ephemeral session identifier for anonymous gameplay. Only its hash is persisted when needed for gameplay records.

## Success Criteria

- **SC-001**: Players can register, log in, refresh, and log out using email/password in under 3 seconds end to end.
- **SC-002**: Players can complete Google or Discord OAuth sign-in in under 5 seconds end to end.
- **SC-003**: Reused refresh tokens are rejected and the corresponding session family is revoked.
- **SC-004**: Auth-sensitive endpoints enforce rate limits after a documented number of attempts from the same actor.
- **SC-005**: Guest sessions can start solo games and join rooms without registering.
- **SC-006**: All backend auth behavior is verified by automated tests including unit, integration, and security tests.

## Assumptions

- OAuth providers use the standard authorization-code flow with client credentials supplied via environment configuration.
- Provider emails are considered verified for account creation and linking.
- Passwords must be at least 12 characters.
- Display names must be between 2 and 32 characters.
- Guest sessions are not allowed to access profile, billing, friends, or registered leaderboards.
- Email verification is out of scope for this phase and will be deferred to a future phase.
