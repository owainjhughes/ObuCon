# Server-Owned Dictionary and Vocabulary Import Design

## Goal

Move dictionary filtering and pagination, plus multi-word vocabulary persistence, into the Go backend without changing the browser-owned Anki connection.

## Dictionary query

`GET /dictionary` keeps `language=ja` and accepts optional `query`, `jlpt`, `page`, and `page_size` parameters. The backend validates the request, searches kanji, hiragana, and meaning, applies the JLPT filter, returns a stable order, and always paginates. Omitted pagination parameters fall back to the first page at the default page size.

The response contains `entries` plus `page`, `page_size`, `total`, and `total_pages`. `page` must be between 1 and 100000, `page_size` must be between 1 and 100, `jlpt` must be between 1 and 5, and `query` is limited to 100 Unicode characters. Search terms are treated literally rather than as SQL wildcard expressions. A page beyond the last one returns no entries.

The React dictionary page requests one page at a time and no longer caches the complete dictionary. It still loads known vocabulary so entries can be marked as known.

## Transactional vocabulary import

`POST /vocab/import` accepts a language and between 1 and 500 entries. Every entry has a required lemma and optional hiragana, meaning, and JLPT level. Request shape and ranges are enforced by the binding tags on the request struct, exactly as the other endpoints in this package do; the handler then trims each entry and collapses duplicate lemmas, with the last occurrence winning, before any database work starts.

Import request bodies are capped at 4 MiB. Oversized requests return 413, validation errors return 400, and persistence failures return a generic 500 response without exposing database details.

The repository loads existing user vocabulary, reuses `GetDictionaryGradeLevels` to infer JLPT levels for the lemmas that still need one, classifies each normalized entry as added, updated, or skipped, and writes all changed entries within one PostgreSQL transaction. A failed write rolls back the complete import. Imported metadata is applied as a JSONB patch so unrelated keys such as conjugation kind are preserved, including flexible non-string values. Missing imported values do not erase existing or concurrently updated values.

The response contains `added`, `updated`, and `skipped` counts.

AnkiConnect remains in the browser because it connects to the user's local Anki process. After reading and mapping mature Anki notes, React sends one request to `/vocab/import`, refreshes vocabulary once, and displays the server counts.

## Testing and rollout

Backend tests cover parameter validation, search filtering and pagination, normalization and deduplication, result classification, commit, and rollback. Frontend tests cover the request parameters and response mapping of the two API modules.

The dictionary response shape changes, so the backend and frontend deploy together. The new vocabulary import requires the backend endpoint; a failed request is shown as an import error and does not fall back to partial per-word writes.
