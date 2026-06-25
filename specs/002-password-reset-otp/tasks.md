---

description: "Task list for password reset with OTP"

---

# Tasks: Password Reset with OTP

**Input**: Design documents from `/specs/002-password-reset-otp/`

**Prerequisites**: plan.md, spec.md

## Phase 1: Email Platform

- [x] T001 Create `backend/internal/platform/email/client.go` with email sender interface
- [x] T002 Create `backend/internal/platform/email/logger.go` development implementation
- [x] T003 Add email config to `backend/internal/config/config.go` and `.env.example`

## Phase 2: OTP Core

- [x] T004 Implement OTP generation in `backend/internal/auth/otp.go`
- [x] T005 Implement OTP Redis storage with TTL and attempt counter
- [x] T006 Add OTP DTOs to `backend/internal/auth/dto.go`
- [x] T007 Add OTP errors to `backend/internal/auth/errors.go`

## Phase 3: Auth Service and Handlers

- [x] T008 Implement `RequestPasswordReset` in auth service
- [x] T009 Implement `ResetPassword` in auth service
- [x] T010 Add `POST /auth/forgot-password` handler
- [x] T011 Add `POST /auth/reset-password` handler
- [x] T012 Wire email client into auth service constructor

## Phase 4: Tests and Contracts

- [x] T013 Unit tests for OTP generation and validation
- [x] T014 Handler tests for forgot/reset endpoints
- [x] T015 Update `backend/openapi/openapi.yaml`
- [x] T016 Run `go test ./...` and `go vet ./...`
