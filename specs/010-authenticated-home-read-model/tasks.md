# Tasks: Authenticated Home Read Model

- [X] T001 Add failing home service tests for success, auth, active-account
  concealment, dependency failure, map bounds, and bounded concurrency.
- [X] T002 Add failing handler and metrics tests for `200`, `401`, `503`, and
  metric registration.
- [X] T003 Implement typed DTOs, errors, metrics, service orchestration, and the
  thin HTTP handler in `backend/internal/home`.
- [X] T004 Mount the registered, rate-limited route and wire dependencies in
  `backend/internal/app` and `backend/cmd/api`.
- [X] T005 Add route-level authentication and rate-limit coverage.
- [X] T006 Add the OpenAPI path and schemas.
- [X] T007 Run formatting, tests, vet, lint, OpenAPI validation, and build gates.
- [X] T008 Harden daily-challenge fallback map selection so test or incomplete
  maps cannot make the authenticated home snapshot unavailable.
