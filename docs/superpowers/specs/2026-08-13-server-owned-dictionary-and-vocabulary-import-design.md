# Server-Owned Dictionary and Vocabulary Import Design

## Goal

Move dictionary filtering and pagination, plus multi-word vocabulary persistence, into the Go backend without changing the browser-owned Anki connection or breaking existing clients during deployment.

## Dictionary query

`GET /dictionary` keeps `language=ja` and accepts optional `query`, `jlpt`, `page`, and `page_size` parameters. When `page` or `page_size` is present, the backend validates the request, searches kanji, hiragana, and meaning, applies the JLPT filter, returns a stable order, and includes pagination metadata. Requests without pagination parameters retain the existing `{ "entries": [...] }` behavior for deployment compatibility.

The paginated response includes `entries` and `pagination` with `page`, `page_size`, `total`, and `total_pages`. `page` must be at least 1, `page_size` must be between 1 and 100, `jlpt` must be between 1 and 5, and `query` is limited to 100 Unicode characters. Search terms are treated literally rather than as SQL wildcard expressions.

The React dictionary page requests one page at a time and no longer caches the complete dictionary. It still loads known vocabulary so entries can be marked as known. If a staggered deployment returns the old unpaginated response, the page temporarily filters and slices that response locally.

## Transactional vocabulary import

`POST /vocab/import` accepts a language and between 1 and 500 entries. Every entry has a required lemma and optional hiragana, meaning, and JLPT level. The complete request is normalized and validated before database work starts. Duplicate lemmas within the request are collapsed, with the last occurrence winning.

Import request bodies are capped at 4 MiB. Oversized requests return 413, validation errors return 400, and persistence failures return a generic 500 response without exposing database details.

The repository loads existing user vocabulary and dictionary JLPT levels, classifies each normalized entry as added, updated, or skipped, and writes all changed entries within one PostgreSQL transaction. A failed write rolls back the complete import. Imported metadata is applied as a JSONB patch so unrelated keys such as conjugation kind are preserved, including flexible non-string values. Missing imported values do not erase existing or concurrently updated values.

The response contains `added`, `updated`, and `skipped` counts.

AnkiConnect remains in the browser because it connects to the user's local Anki process. After reading and mapping mature Anki notes, React sends one request to `/vocab/import`, refreshes vocabulary once, and displays the server counts.

## Testing and rollout

Backend tests cover parameter validation, search filtering/pagination, normalization/deduplication, result classification, commit, and rollback. Frontend tests cover compatibility parsing of new and old dictionary responses and construction of the batch import payload. Existing endpoints remain available.

The backend can deploy before the frontend with no behavior change for old clients. The updated frontend can also tolerate an old backend's dictionary response. The new vocabulary import requires the backend endpoint; a failed request is shown as an import error and does not fall back to partial per-word writes.
