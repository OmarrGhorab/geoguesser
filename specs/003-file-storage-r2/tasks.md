---

description: "Task list for file storage with R2"

---

# Tasks: File Storage with Cloudflare R2

**Input**: Design documents from `/specs/003-file-storage-r2/`

**Prerequisites**: plan.md, spec.md

## Phase 1: Database and Platform

- [x] T001 Add Goose migration `backend/migrations/00003_uploads_files.sql`
- [x] T002 Create `backend/internal/platform/storage/storage.go` interface
- [x] T003 Implement R2/S3 adapter in `backend/internal/platform/storage/r2.go`
- [x] T004 Implement local filesystem adapter in `backend/internal/platform/storage/local.go`
- [x] T005 Add R2 config to `backend/internal/config/config.go` and `.env.example`

## Phase 2: Uploads Feature

- [x] T006 Create `backend/internal/uploads/model.go`
- [x] T007 Create `backend/internal/uploads/dto.go`
- [x] T008 Create `backend/internal/uploads/errors.go`
- [x] T009 Create `backend/internal/uploads/repository.go`
- [x] T010 Create `backend/internal/uploads/service.go` with validation
- [x] T011 Create `backend/internal/uploads/handler.go`

## Phase 3: Routes and Tests

- [x] T012 Wire uploads handler into `backend/internal/app/routes.go`
- [x] T013 Unit tests for file validation
- [x] T014 Handler tests for upload endpoints
- [x] T015 Update `backend/openapi/openapi.yaml`
- [x] T016 Run `go test ./...` and `go vet ./...`
