# Feature Specification: File Storage with Cloudflare R2

**Feature Branch**: `003-file-storage-r2`

**Created**: 2026-06-26

**Status**: Approved

**Input**: User description: "setup R2 cloudflare for uploads"

## User Scenarios & Testing

### User Story 1 - Create Upload (Priority: P1)

A registered user can request permission to upload a file. The backend returns a presigned upload URL and an upload id. The client uploads directly to R2.

**Why this priority**: Direct-to-R2 uploads keep large files out of the API and reduce bandwidth.

**Independent Test**: An authenticated user can request an upload URL for a valid file.

**Acceptance Scenarios**:

1. **Given** an authenticated user and valid file metadata, **When** they create an upload, **Then** they receive a presigned URL and upload id.
2. **Given** a guest session, **When** they attempt to create an upload, **Then** the request is rejected.
3. **Given** an unsupported MIME type or oversized file, **When** they attempt to create an upload, **Then** the request is rejected.

---

### User Story 2 - Complete Upload (Priority: P1)

After the client uploads the file to R2, it notifies the backend. The backend validates the object exists and records the file metadata.

**Why this priority**: Lets the backend know the upload succeeded before the file is used.

**Independent Test**: A user can complete an upload after a successful R2 PUT.

**Acceptance Scenarios**:

1. **Given** a successful R2 upload, **When** the client calls complete, **Then** the file is recorded.
2. **Given** an upload id for a missing object, **When** the client calls complete, **Then** the request is rejected.

---

### User Story 3 - Signed Download URL (Priority: P1)

An authorized user can request a time-limited signed URL to download a private file.

**Why this priority**: Private files should not be publicly readable.

**Independent Test**: An authenticated user can request a signed URL for their own file.

**Acceptance Scenarios**:

1. **Given** a recorded file, **When** the owner requests a signed URL, **Then** a short-lived URL is returned.
2. **Given** another user's file, **When** a user requests a signed URL, **Then** the request is rejected.

## Requirements

### Functional Requirements

- **FR-001**: The system MUST support direct uploads to Cloudflare R2 using S3-compatible presigned URLs.
- **FR-002**: The system MUST validate file size, MIME type, and extension before generating an upload URL.
- **FR-003**: The system MUST restrict upload creation to registered users.
- **FR-004**: The system MUST generate unique file ids and object keys.
- **FR-005**: The system MUST validate object existence on upload completion.
- **FR-006**: The system MUST provide signed download URLs with configurable expiration.
- **FR-007**: The system MUST prevent users from accessing other users' private files.
- **FR-008**: The system MUST never trust client-provided file paths or names as storage keys.

### Key Entities

- **Upload**: A pending upload record with id, owner, MIME type, size, and expiry.
- **File**: A completed file record with id, owner, storage key, MIME type, size, and created timestamp.

## Success Criteria

- **SC-001**: Authenticated users can create R2 presigned upload URLs.
- **SC-002**: Upload size is capped at 10 MB by default.
- **SC-003**: Signed download URLs expire within 15 minutes by default.
- **SC-004**: File access is restricted to the uploading user unless explicitly public.

## Assumptions

- R2 credentials (account id, access key, secret key, bucket) are provided via environment variables.
- Files are private by default.
- Upload completion is synchronous; background verification is out of scope.
