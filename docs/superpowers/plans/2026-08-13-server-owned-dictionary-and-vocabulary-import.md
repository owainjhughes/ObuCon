# Server-Owned Dictionary and Vocabulary Import Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add server-side dictionary filtering/pagination and an atomic multi-word vocabulary import, then migrate the React flows to use them.

**Architecture:** Extend the existing analysis handler/service/repository path with value objects for dictionary queries and imports. PostgreSQL owns filtering, pagination, and the import transaction; React owns interaction and the local AnkiConnect adapter.

**Tech Stack:** Go 1.25+, Gin, GORM/PostgreSQL, React 19, TypeScript, Axios, Jest/React Scripts.

**Spec:** `docs/superpowers/specs/2026-08-13-server-owned-dictionary-and-vocabulary-import-design.md`

## Global Constraints

- Preserve existing unpaginated `GET /dictionary` behavior when pagination is not requested.
- Accept at most 500 vocabulary import entries and perform all changed writes in one transaction.
- Treat dictionary search metacharacters literally.
- Keep AnkiConnect browser-side.
- Add no code comments unless a non-obvious constraint cannot be expressed in code; any new comment must be EN+JP.

---

### Task 1: Dictionary query contract

**Files:** `backend/internal/analysis/repository.go`, `service.go`, `handler.go`, and new `dictionary_test.go`.

- [x] Write failing tests for validation, literal search, pagination, and ordering.
- [x] Run the focused tests and verify the expected red state.
- [x] Implement query parsing, service delegation, and repository queries while keeping the old list path.
- [x] Run focused tests until green.

### Task 2: Transactional vocabulary import

**Files:** `backend/internal/analysis/repository.go`, `service.go`, `handler.go`, `backend/cmd/server/main.go`, and new `import_test.go`.

- [x] Write failing tests for normalization, deduplication, validation, classification, commit, and rollback.
- [x] Run the focused tests and verify the expected red state.
- [x] Implement validation before repository access.
- [x] Implement transactional metadata merge, grade inference, classification, and upsert.
- [x] Register `POST /vocab/import` and run all backend tests.

### Task 3: Frontend adapters

**Files:** new `frontend/src/api/dictionary.ts`, `dictionary.test.ts`, `vocabulary.ts`, and `vocabulary.test.ts`.

- [x] Write failing tests for paginated/new and unpaginated/old dictionary responses plus batch import payload construction.
- [x] Run focused tests and verify the expected red state.
- [x] Implement the pure adapters and run focused tests until green.

### Task 4: React migrations

**Files:** `frontend/src/pages/Dictionary.tsx` and `Vocab.tsx`.

- [x] Move dictionary filtering and pagination requests to the server with old-backend response compatibility.
- [x] Replace per-entry Anki persistence with one batch request and display server counts.
- [x] Run frontend tests and production build.

### Task 5: Full verification

- [x] Format changed Go files.
- [x] Run Go tests with coverage and Go vet.
- [x] Run frontend tests and production build.
- [x] Run `git diff --check` and review the complete diff.
