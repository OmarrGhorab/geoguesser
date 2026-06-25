# Feature Specification: Password Reset with OTP

**Feature Branch**: `002-password-reset-otp`

**Created**: 2026-06-26

**Status**: Approved

**Input**: User description: "implement forget reset password with otp"

## User Scenarios & Testing

### User Story 1 - Request Password Reset OTP (Priority: P1)

A registered player who forgot their password can request a one-time password (OTP) to be sent to their email address. The OTP is short-lived and rate limited.

**Why this priority**: This is the entry point of the password recovery flow.

**Independent Test**: A registered user can request an OTP and receive a confirmation that the OTP was sent.

**Acceptance Scenarios**:

1. **Given** a registered email address, **When** the player requests a password reset, **Then** an OTP is generated and sent to the email.
2. **Given** an unregistered email address, **When** the player requests a password reset, **Then** the response still indicates the OTP was sent to prevent email enumeration.
3. **Given** repeated reset requests, **When** the rate limit is exceeded, **Then** subsequent requests are rejected until the window resets.

---

### User Story 2 - Reset Password with OTP (Priority: P1)

A registered player with a valid OTP can set a new password. The OTP is consumed and cannot be reused.

**Why this priority**: Completes the password recovery flow.

**Independent Test**: A player can request an OTP and immediately use it to set a new password and log in.

**Acceptance Scenarios**:

1. **Given** a valid OTP and a new password that meets policy, **When** the player resets the password, **Then** the password is updated and the OTP is invalidated.
2. **Given** an invalid or expired OTP, **When** the player attempts to reset, **Then** the request is rejected.
3. **Given** a new password that is too short, **When** the player attempts to reset, **Then** the request is rejected with a validation error.

---

### User Story 3 - Email Delivery Abstraction (Priority: P2)

The backend uses a swappable email client so OTPs can be logged in development and sent via a real provider in production without changing feature code.

**Why this priority**: Enables local testing and future provider integration.

**Independent Test**: In development mode, requesting an OTP logs the OTP through the configured email client.

**Acceptance Scenarios**:

1. **Given** development configuration, **When** an OTP is sent, **Then** it is logged via structured logs without exposing secrets.
2. **Given** production configuration with a provider, **When** an OTP is sent, **Then** it is delivered through that provider.

## Requirements

### Functional Requirements

- **FR-001**: The system MUST allow registered players to request a password reset OTP via email.
- **FR-002**: The system MUST generate a cryptographically secure, short numeric/alphanumeric OTP.
- **FR-003**: The system MUST store OTPs in Redis with a short TTL.
- **FR-004**: The system MUST rate limit OTP requests per email address.
- **FR-005**: The system MUST allow players to reset their password using a valid OTP.
- **FR-006**: The system MUST invalidate the OTP after a successful reset.
- **FR-007**: The system MUST enforce the existing password policy on the new password.
- **FR-008**: The system MUST NOT reveal whether an email is registered when requesting an OTP.
- **FR-009**: The system MUST use an interface-driven email client for OTP delivery.
- **FR-010**: The email client MUST redact the OTP from general logs; only the email abstraction should see it.

### Key Entities

- **Password Reset OTP**: A short-lived code stored in Redis keyed by email hash, with an attempt counter.
- **Email Client**: An interface for sending transactional emails. The logging implementation is used in development.

## Success Criteria

- **SC-001**: Players can request and receive an OTP and reset their password end to end.
- **SC-002**: OTPs expire after 10 minutes by default.
- **SC-003**: OTP requests are rate limited to 3 attempts per 10 minutes per email.
- **SC-004**: Invalid or expired OTPs are rejected.
- **SC-005**: Email enumeration is prevented in forgot-password responses.

## Assumptions

- OTPs are 6-digit numeric codes.
- OTP delivery in development is logged to stdout; production uses a real provider passed via config.
- Existing password policy requires at least 12 characters.
