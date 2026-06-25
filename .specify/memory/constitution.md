<!--
Sync Impact Report
Version change: template -> 1.0.0
Modified principles:
- PRINCIPLE_1_NAME -> I. Code Quality and Architecture
- PRINCIPLE_2_NAME -> II. Testing Standards and Quality Gates
- PRINCIPLE_3_NAME -> III. User Experience Consistency
- PRINCIPLE_4_NAME -> IV. Performance Budgets and Measurement
- PRINCIPLE_5_NAME -> V. Specification Traceability and Operational Readiness
Added sections:
- Project Standards
- Delivery Workflow and Quality Gates
Removed sections:
- SECTION_2_NAME placeholder
- SECTION_3_NAME placeholder
Templates requiring updates:
- Updated: .specify/templates/plan-template.md
- Updated: .specify/templates/spec-template.md
- Updated: .specify/templates/tasks-template.md
- Updated: .specify/templates/checklist-template.md
- Reviewed: .specify/templates/commands/*.md (no command templates present)
Runtime guidance:
- Updated: README.md
- Updated: AGENTS.md
- Reviewed: docs/phase-9-project-setup.md
Follow-up TODOs: None
-->
# GeoGuess Constitution

## Core Principles

### I. Code Quality and Architecture
Production code MUST be typed, cohesive, and placed in the established workspace
boundaries: `client/` for the Next.js App Router UI, `backend/` for the Go API,
and `docs/` for planning artifacts. Backend code MUST keep executable setup in
`cmd/api`, application packages under `internal/`, infrastructure adapters under
`internal/platform/`, and database changes in Goose migrations; production code
MUST NOT use GORM AutoMigrate. Frontend code MUST follow the installed Next.js
16 guidance in `client/node_modules/next/dist/docs/` before changing
framework-sensitive APIs, and MUST use client components only for browser state
or browser-only interactions. New abstractions MUST remove meaningful duplication
or isolate a real boundary.

Rationale: consistent boundaries reduce coupling, keep reviews focused, and make
future feature work safer.

### II. Testing Standards and Quality Gates
Every behavior change MUST include automated verification at the closest useful
level: unit tests for pure logic, contract tests for API shape changes,
integration tests for service/data interactions, and browser-flow tests or
recorded manual evidence for UI journeys that require browser APIs. Bug fixes
MUST include a regression test that fails before the fix when the failure can be
reproduced locally. Required merge gates are backend `go test ./...`, frontend
`pnpm lint`, frontend `pnpm typecheck`, frontend `pnpm build`, and every
targeted test introduced for the feature. Any omitted gate MUST be justified in
the feature plan with the exact blocker and residual risk.

Rationale: a full-stack game depends on scoring logic, API contracts, and
browser behavior staying correct together.

### III. User Experience Consistency
User-facing changes MUST preserve the GeoGuess interaction model, visual system,
and localized routing conventions. Localized screens MUST source visible UI copy
from message catalogs and MUST support both `en` and `ar`, including RTL layout
behavior for Arabic. Interactive surfaces MUST define loading, empty, error,
disabled, and success states before implementation is complete. Components MUST
reuse the existing Tailwind v4, shadcn/ui, and Radix patterns before introducing
new primitives. Keyboard focus, semantic markup, and accessible names are
required for actionable controls.

Rationale: the game needs to feel coherent across modes, locales, and repeated
rounds rather than like separate feature prototypes.

### IV. Performance Budgets and Measurement
Each feature plan MUST state measurable performance budgets before
implementation: API latency or throughput for backend work, render and bundle
impact for frontend work, and query/storage cost for data work. Implementations
MUST avoid duplicate network calls, unbounded database queries, avoidable
client-only rendering, oversized static assets, and main-thread work that blocks
gameplay interactions. Any accepted regression MUST include measurement evidence
and an explicit tradeoff in the plan.

Rationale: map, Street View, multiplayer, and leaderboard workflows are
latency-sensitive, and performance regressions are expensive to recover after
the user experience depends on them.

### V. Specification Traceability and Operational Readiness
Every feature MUST maintain a traceable chain from spec to plan to tasks, or
update the relevant planning document when Spec Kit is not used. Tasks MUST map
to user stories, acceptance scenarios, and quality gates. Backend endpoint
changes MUST update OpenAPI contracts, data changes MUST include migration
tasks, and runtime behavior changes MUST cover configuration, logging, security,
and readiness/health implications. Features are not complete until verification
steps and operational notes are recorded.

Rationale: traceability keeps product intent, implementation work, and release
readiness aligned across the frontend, backend, and infrastructure.

## Project Standards

- The supported stack is Next.js 16.2.9 App Router, React 19, TypeScript,
  Tailwind CSS v4, shadcn/ui, Radix, next-intl, Zustand for UI preferences,
  Go 1.24+ with Chi Router, PostgreSQL, GORM, Goose, Redis, Docker Compose, and
  GitHub Actions.
- Frontend package management MUST use the pinned pnpm version declared in
  `package.json` and `client/package.json`. On this Windows workspace, commands
  MAY use `npx pnpm@10.24.0` when a global pnpm binary is unavailable.
- Server-side data access in the frontend MUST use native `fetch()` through
  server-only modules. Zustand MUST NOT hold canonical server state.
- API contracts MUST stay in `backend/openapi/openapi.yaml` or generated
  contract artifacts referenced by the feature plan.
- New configuration MUST be documented in the relevant `.env.example` file.
  Secrets MUST NOT be committed.
- Logs and telemetry MUST avoid raw secrets, auth tokens, precise private
  location data, and personally identifying data unless the feature plan records
  a compliant retention and redaction strategy.

## Delivery Workflow and Quality Gates

1. A feature specification MUST define independently testable user stories,
   measurable success criteria, edge cases, and any localization or accessibility
   expectations that affect users.
2. A feature plan MUST pass the Constitution Check before implementation. The
   check MUST cover architecture boundaries, required tests, UX states,
   localization/RTL impact, performance budgets, data/API contract changes, and
   operational readiness.
3. A task list MUST include concrete file paths and verification tasks for every
   required gate. Tests that protect required behavior MUST be scheduled before
   implementation tasks for that behavior.
4. Complexity, gate omissions, or performance regressions MUST be recorded in
   the plan's Complexity Tracking section with a rejected simpler alternative.
5. Before merge or release, reviewers MUST confirm that the implemented behavior
   matches the spec, all required gates have evidence, and documentation or
   contract updates are present.

## Governance

This constitution supersedes conflicting local practices. More specific project
documents may add stricter requirements, but they MUST NOT weaken these
principles without a constitution amendment.

Amendments MUST include a rationale, a semantic version decision, an updated
Sync Impact Report, and matching changes to affected Spec Kit templates or
runtime guidance. The change is ratified when the updated constitution and
propagated artifacts are committed together.

Versioning policy:
- MAJOR: principle removals, incompatible governance changes, or redefinitions
  that invalidate existing approved plans.
- MINOR: new principles, new required sections, or materially expanded quality
  gates.
- PATCH: wording clarifications, typo fixes, and non-semantic refinements.

Compliance review is mandatory for every feature plan and code review. If a
feature cannot satisfy a principle, implementation MUST stop until the exception
is documented, justified, and accepted in the plan.

**Version**: 1.0.0 | **Ratified**: 2026-06-25 | **Last Amended**: 2026-06-25
