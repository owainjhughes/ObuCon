# Test Cases Overview

All 41 tests passed on 2026-04-25.

---

## `internal/auth/service_test.go`

Tests the core authentication service in isolation — JWT generation/validation and bcrypt password hashing. These are critical security primitives used by every protected route, so correctness and rejection of invalid inputs must be verified independently of the HTTP layer.

| Test | Description | Result |
|------|-------------|--------|
| `TestGenerateAndValidateToken_RoundTrip` | Token generated for a user ID round-trips back to the same ID. | PASS |
| `TestValidateToken_WrongSecret` | Token signed with secret A is rejected when validated with secret B. | PASS |
| `TestValidateToken_Malformed` | Malformed and empty strings are rejected. | PASS |
| `TestValidateToken_Expired` | Token with a past `exp` claim is rejected. | PASS |
| `TestBcryptHashAndCompare` | Correct password matches hash; wrong password does not; two hashes of the same password differ. | PASS |

---

## `internal/auth/handler_test.go`

Tests the HTTP handlers and auth middleware for the authentication routes (`/register`, `/login`, `/logout`, `/me`). Uses `httptest` to exercise request binding, cookie management, and the Bearer/cookie token extraction logic without a real server or database.

| Test | Description | Result |
|------|-------------|--------|
| `TestRegister_BindingErrors` | 7 invalid request bodies each return HTTP 400. | PASS |
| `TestLogin_BindingErrors` | 4 invalid request bodies each return HTTP 400. | PASS |
| `TestLogout_ClearsCookie` | Response sets auth cookie with empty value and negative `MaxAge`. | PASS |
| `TestGetMe_Unauthorized` | `GET /me` with no user ID in context returns HTTP 401. | PASS |
| `TestUpdateMe_Unauthorized` | `PUT /me` with no user ID in context returns HTTP 401. | PASS |
| `TestUpdateMe_BindingErrorWithUserID` | `PUT /me` with valid user ID but malformed body returns HTTP 400. | PASS |
| `TestAuthMiddleware_RejectsMissingHeaderAndCookie` | No header or cookie returns HTTP 401. | PASS |
| `TestAuthMiddleware_RejectsMalformedAuthorizationHeader` | Wrong scheme, partial bearer, and basic auth all return HTTP 401. | PASS |
| `TestAuthMiddleware_RejectsInvalidToken` | `Bearer not-a-jwt` returns HTTP 401. | PASS |
| `TestAuthMiddleware_AcceptsValidBearerToken` | Valid bearer token passes; user ID is stored in context. | PASS |
| `TestAuthMiddleware_AcceptsValidCookieToken` | Valid cookie token passes; user ID is stored in context. | PASS |
| `TestAuthMiddleware_EmptyCookieFallsThroughToUnauthorized` | Empty cookie value returns HTTP 401. | PASS |
| `TestAuthMiddleware_HeaderTakesPrecedenceOverCookie` | When both present, the header token's user ID wins. | PASS |

---

## `internal/config/config_test.go`

Tests the environment variable helper functions used to load application configuration at startup. Correct fallback behaviour is important for both local development and production deployments where some variables may be absent.

| Test | Description | Result |
|------|-------------|--------|
| `TestGetEnv` | Returns set value; falls back when unset; returns empty string when explicitly set to empty. | PASS |
| `TestGetEnvBool` | 8 cases: unset (fallback), `"true"`, `"false"`, `"1"`, `"0"`, and invalid values (fallback). | PASS |
| `TestSplitCSV` | 7 cases: simple CSV, whitespace trimming, empty segments, blank/whitespace-only input, and single value. | PASS |

---

## `internal/helpers/auth_context_test.go`

Tests the `UserIDFromContext` helper that extracts the authenticated user's ID from a Gin request context. This is used by every handler that requires an authenticated user, so type safety and edge cases (absent key, wrong type, zero value) must be handled correctly.

| Test | Description | Result |
|------|-------------|--------|
| `TestUserIDFromContext_Present` | `uint(42)` in context is returned with `ok=true`. | PASS |
| `TestUserIDFromContext_Absent` | Nothing stored returns `0, false`. | PASS |
| `TestUserIDFromContext_WrongType` | string, int, int64, float64, and nil all return `0, false`. | PASS |
| `TestUserIDFromContext_ZeroValueUint` | `uint(0)` returns `0, true` — a valid stored zero is distinguishable from absent. | PASS |

---

## `internal/helpers/read_textfile_test.go`

Tests the file upload text extraction pipeline — normalising line endings and whitespace, and extracting text from multipart file headers. This guards the input boundary before text is passed to the Japanese tokeniser for analysis.

| Test | Description | Result |
|------|-------------|--------|
| `TestNormalizeTextForAnalysis` | 6 cases: CRLF/CR→LF, trailing whitespace, consecutive blank line cap, leading/trailing trim, passthrough, empty. | PASS |
| `TestExtractTextFromFileHeader_UnsupportedExtension` | `.exe` file returns an "unsupported file type" error. | PASS |
| `TestExtractTextFromFileHeader_NilHeader` | `nil` header returns an error. | PASS |
| `TestExtractTextFromFileHeader_Empty` | Zero-byte file returns an "empty" error. | PASS |
| `TestExtractTextFromFileHeader_PlainText` | `.txt` file: filename, normalised text, and character count are correct. | PASS |
| `TestExtractTextFromFileHeader_MarkdownAccepted` | `.md` file is accepted and content extracted. | PASS |
| `TestExtractTextFromFileHeader_NonUTF8Rejected` | Non-UTF-8 bytes return a "UTF-8" error. | PASS |

---

## `internal/lang/ja/tokenizer_test.go`

Tests the Japanese tokeniser built on Kagome (IPA dictionary). This is the core NLP component of the application — it segments Japanese text into tokens and classifies them by script type, feeding the lemma extraction used in analysis.

| Test | Description | Result |
|------|-------------|--------|
| `TestIsNumericToken` | 7 cases: ASCII digits, full-width digits, empty, and mixed alphanumeric. | PASS |
| `TestIsKatakanaToken` | 9 cases: pure katakana, mixed scripts, hiragana, roman, numeric, and prolonged vowel mark. | PASS |
| `TestIsRomanToken` | 9 cases: lowercase, mixed case, apostrophes, alphanumeric, digits-only, katakana, spaces/punctuation. | PASS |
| `TestNewTokenizer` | `NewTokenizer()` returns a non-nil tokenizer without error. | PASS |
| `TestTokenize_Empty` | Empty string returns zero tokens. | PASS |
| `TestTokenize_FiltersPunctuationAndBosEos` | Japanese sentence produces no BOS/EOS markers, no punctuation, no empty lemmas. | PASS |
| `TestTokenize_Deterministic` | Same input tokenised twice yields identical results. | PASS |
| `TestTokenize_KatakanaFlagSet` | `"コーヒー"` produces a token with `IsKatakana=true`. | PASS |

---

## `internal/analysis/service_test.go`

Tests the `UniqueLemmas` function that deduplicates and orders lemmas from tokeniser output before returning them to the user. Correct deduplication and insertion-order preservation are essential for consistent, reproducible analysis results.

| Test | Description | Result |
|------|-------------|--------|
| `TestUniqueLemmas` | 7 cases: nil input, single token, duplicate deduplication, surface/lemma differing (lemma first), empty lemma skipped, empty surface+lemma skipped, mixed scenario. | PASS |
