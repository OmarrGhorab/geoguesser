# Implementation Plan: File Storage with Cloudflare R2

**Branch**: `003-file-storage-r2` | **Date**: 2026-06-26 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/003-file-storage-r2/spec.md`

## Summary

Add Cloudflare R2-backed file uploads. Implement a storage platform adapter using the AWS SDK for S3, upload/file records in PostgreSQL, and REST endpoints for creating uploads, completing uploads, and generating signed download URLs.

## Technical Context

**Language/Version**: Go 1.25

**Primary Dependencies**: AWS SDK for Go v2 (S3), Chi, GORM

**Storage**: PostgreSQL for upload/file metadata, Cloudflare R2 for object storage

**Testing**: `go test ./...`

**Performance Goals**: Presigned URL generation p95 under 100ms

**Constraints**: Max upload size 10 MB; signed URLs expire in 15 minutes

## Constitution Check

- **Architecture boundaries**: PASS. Work stays in `backend/internal/platform/storage` and a new `backend/internal/uploads` feature package. Database changes use Goose migrations.
- **Testing gates**: PASS. Unit tests for validation and storage adapter; handler tests for endpoints.
- **Contracts and data**: PASS. OpenAPI will be updated. Goose migration will add `uploads` and `files` tables.
- **Operational readiness**: PASS. R2 credentials documented in `.env.example`.

## Project Structure

```text
backend/
├── internal/
│   ├── platform/
│   │   └── storage/
│   │       ├── storage.go     # storage interface
│   │       ├── r2.go          # R2/S3 implementation
│   │       └── local.go       # local filesystem implementation for tests
│   └── uploads/
│       ├── handler.go
│       ├── service.go
│       ├── repository.go
│       ├── model.go
│       ├── dto.go
│       └── errors.go
├── migrations/00003_uploads_files.sql
└── openapi/openapi.yaml
```

## Complexity Tracking

No constitution violations. R2 is S3-compatible, so the AWS SDK is the standard adapter.
