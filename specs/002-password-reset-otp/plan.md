# Implementation Plan: Password Reset with OTP

**Branch**: `002-password-reset-otp` | **Date**: 2026-06-26 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/002-password-reset-otp/spec.md`

## Summary

Add password reset via OTP to the auth system. Implement email abstraction, OTP generation/storage in Redis, rate limiting, and reset endpoint. Keep the backend modular and testable.

## Technical Context

**Language/Version**: Go 1.25

**Primary Dependencies**: Chi, Redis, golang.org/x/crypto

**Storage**: Redis for OTP codes and attempt counters

**Testing**: `go test ./...`

**Project Type**: web-service backend

**Performance Goals**: OTP endpoints p95 latency under 200ms excluding email provider latency

**Constraints**: OTPs must not be logged in general request logs; email client owns delivery logging

## Constitution Check

- **Architecture boundaries**: PASS. Work stays in `backend/internal/auth` and `backend/internal/platform/email`. No GORM AutoMigrate.
- **Testing gates**: PASS. Unit tests for OTP logic and handler tests for endpoints included.
- **Performance budgets**: PASS. Redis lookups are O(1).
- **Contracts and data**: PASS. OpenAPI will be updated. No schema changes needed.
- **Operational readiness**: PASS. New config keys documented in `.env.example`.

## Project Structure

```text
backend/
├── internal/
│   ├── auth/
│   │   ├── handler.go         # add forgot/reset endpoints
│   │   ├── service.go         # add OTP logic
│   │   ├── otp.go             # OTP generation and storage
│   │   ├── dto.go             # add request DTOs
│   │   └── errors.go          # add OTP errors
│   └── platform/
│       └── email/
│           ├── client.go      # email interface
│           └── logger.go      # development logging implementation
├── internal/config/config.go  # email config
└── openapi/openapi.yaml       # new auth endpoints
```

## Complexity Tracking

No constitution violations. Email abstraction adds a small interface but is required to keep local development usable without a real provider.
