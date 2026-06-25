# Implementation Plan: [FEATURE]

**Branch**: `[###-feature-name]` | **Date**: [DATE] | **Spec**: [link]

**Input**: Feature specification from `/specs/[###-feature-name]/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

[Extract from feature spec: primary requirement + technical approach from research]

## Technical Context

<!--
  ACTION REQUIRED: Replace the content in this section with the technical details
  for the project. The structure here is presented in advisory capacity to guide
  the iteration process.
-->

**Language/Version**: [e.g., Python 3.11, Swift 5.9, Rust 1.75 or NEEDS CLARIFICATION]

**Primary Dependencies**: [e.g., FastAPI, UIKit, LLVM or NEEDS CLARIFICATION]

**Storage**: [if applicable, e.g., PostgreSQL, CoreData, files or N/A]

**Testing**: [e.g., pytest, XCTest, cargo test or NEEDS CLARIFICATION]

**Target Platform**: [e.g., Linux server, iOS 15+, WASM or NEEDS CLARIFICATION]

**Project Type**: [e.g., library/cli/web-service/mobile-app/compiler/desktop-app or NEEDS CLARIFICATION]

**Performance Goals**: [domain-specific, e.g., 1000 req/s, 10k lines/sec, 60 fps or NEEDS CLARIFICATION]

**Constraints**: [domain-specific, e.g., <200ms p95, <100MB memory, offline-capable or NEEDS CLARIFICATION]

**Scale/Scope**: [domain-specific, e.g., 10k users, 1M LOC, 50 screens or NEEDS CLARIFICATION]

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Document PASS/FAIL for each gate. Any FAIL requires an entry in Complexity
Tracking with the exact blocker, residual risk, and rejected simpler alternative.

- **Architecture boundaries**: Feature keeps frontend work in `client/`,
  backend work in `backend/`, planning in `docs/` or `specs/`, backend packages
  under `internal/`, executable setup in `cmd/api`, infrastructure adapters in
  `internal/platform/`, and database changes in Goose migrations. Production
  code does not use GORM AutoMigrate.
- **Framework guidance**: Frontend plan identifies the relevant installed
  Next.js guide under `client/node_modules/next/dist/docs/` before changing
  framework-sensitive APIs.
- **Testing gates**: Plan lists required unit, contract, integration, and UI
  verification for the behavior changed, plus the required commands:
  backend `go test ./...`, frontend `pnpm lint`, frontend `pnpm typecheck`,
  frontend `pnpm build`, and targeted feature tests.
- **UX consistency**: Plan covers loading, empty, error, disabled, and success
  states for interactive surfaces; keyboard focus, semantic markup, and
  accessible names for controls; and reuse of Tailwind v4, shadcn/ui, and Radix
  patterns before adding primitives.
- **Localization and RTL**: User-facing copy is planned through message catalogs
  for `en` and `ar`, with Arabic RTL layout impact documented or marked N/A.
- **Performance budgets**: Plan states measurable API latency or throughput,
  frontend render/bundle impact, data query/storage costs, and measurement
  method. Any accepted regression is explicitly justified.
- **Contracts and data**: API changes update `backend/openapi/openapi.yaml` or
  referenced generated contracts. Data changes include Goose migration tasks.
- **Operational readiness**: Plan covers configuration, `.env.example` changes,
  logging/redaction, security implications, and health/readiness impact.

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)
<!--
  ACTION REQUIRED: Replace the placeholder tree below with the concrete layout
  for this feature. Delete unused options and expand the chosen structure with
  real paths (e.g., apps/admin, packages/something). The delivered plan must
  not include Option labels.
-->

```text
# [REMOVE IF UNUSED] Option 1: Single project (DEFAULT)
src/
├── models/
├── services/
├── cli/
└── lib/

tests/
├── contract/
├── integration/
└── unit/

# [REMOVE IF UNUSED] Option 2: GeoGuess web application
backend/
├── cmd/api/
├── internal/
│   ├── app/
│   ├── config/
│   ├── health/
│   ├── middleware/
│   └── platform/
├── migrations/
└── openapi/

client/
├── app/
├── components/
├── features/
├── lib/
├── messages/
└── stores/

docs/
└── [planning and architecture docs]

# [REMOVE IF UNUSED] Option 3: Mobile + API (when "iOS/Android" detected)
api/
└── [same as backend above]

ios/ or android/
└── [platform-specific structure: feature modules, UI flows, platform tests]
```

**Structure Decision**: [Document the selected structure and reference the real
directories captured above]

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |
