# PMVHaven Filename-to-Thumbnail Matching Spec

## Goal

Match local unmatched PMV video files to PMVHaven scenes by filename, extract a reliable thumbnail URL from PMVHaven HTML, and attach that thumbnail to the matched scene so the video shows the correct cover image in XBVR.

## Current State (as implemented)

### Entry point

- Single-file API exists: `POST /api/task/pmv-match` in `pkg/api/tasks.go`.
- Handler calls `tasks.MatchPMVFile(file_id, dry_run)` in `pkg/tasks/pmv_match.go`.

### Existing matching pipeline

1. Load `models.File` by `file_id`.
2. Reject if already matched (`scene_id != 0`).
3. Normalize filename to query with `normalizePMVQuery`.
4. Search PMVHaven with `scrape.SearchPMVHaven(query, limit=5)`.
5. Parse candidates from search HTML using `ParsePMVHavenSearchHTML`.
6. Rank candidates:
- Baseline token-overlap ranking.
- Optional OpenAI rerank if `OPENAI_API_KEY` is set.
7. Auto-link only when top confidence >= `0.85`.
8. Persist scene via `SceneCreateUpdateFromExternal`, then link file to created scene.

### Existing PMVHaven parsing behavior

- Candidate fields include `ID`, `Title`, `SceneURL`, `ThumbnailURL`.
- Thumbnails are currently extracted from search result HTML (anchors, cards, JSON-LD).
- Canonical PMVHaven scene URL and PMV ID extraction already exist.

### Existing persistence behavior

- Matched scene is created as:
- `scene_id = pmvhaven-{pmv_id}`
- `scraper_id = pmvhaven`
- `site/studio = PMVHaven`
- `cover_url` and `images` are populated from `ext.Covers` (first entry becomes `cover_url`).

## Gap vs requested behavior

Requested flow: read video folder filenames -> parse names -> search PMVHaven -> fetch result HTML -> parse thumbnail -> match thumbnail to video.

Current gap:

1. Matching is only exposed for a single `file_id`, not a folder/batch flow.
2. Thumbnail source is mainly search-page HTML; no dedicated scene-page fetch to improve thumbnail reliability.
3. No explicit batch reporting for unmatched files processed from library folders.

## Proposed Feature Scope

## Phase 1 (MVP-hardening, low risk)

Enhance candidate enrichment to fetch PMVHaven scene page HTML per candidate (with small cap) and extract thumbnail using scene-page selectors/fallbacks:

- `meta[property="og:image"]`
- `meta[name="twitter:image"]`
- JSON-LD `thumbnailUrl` / `image`
- `<video poster>`

Rules:

- If search thumbnail is missing or low-confidence, use scene-page thumbnail.
- Keep first valid absolute URL.
- Preserve existing ranking and autolink threshold.

Expected result: better thumbnail quality without changing API contract.

## Phase 2 (folder-level matching flow)

Add a batch task for unmatched video files (from existing scanned library records):

- New endpoint: `POST /api/task/pmv-match-unmatched`
- Request:
- `dry_run` (bool)
- `limit` (int, default 50, max 500)
- `volume_id` (optional)
- `path_prefix` (optional)
- Response:
- totals (`scanned`, `matched`, `skipped_already_matched`, `below_threshold`, `errors`)
- per-file result list (same shape as current `PMVMatchResult` + error field)

File source for batch:

- Query `files where type='video' and scene_id=0`.
- Optional filters by volume or path prefix.

This avoids direct filesystem traversal in the PMV task and reuses existing volume scan DB state.

## Detailed Design

### New scraper function(s)

In `pkg/scrape/pmvhaven.go`:

- `func EnrichPMVHavenCandidateThumbnail(c PMVHavenCandidate) (PMVHavenCandidate, error)`
- `func ParsePMVHavenSceneHTMLForThumbnail(htmlBody string) string`

Implementation notes:

- Reuse resty client settings, `User-Agent`, `SetupRestyRequest("pmvhaven-scraper", req)`.
- Timeout 25s, retry count 2.
- Avoid failing whole match on enrichment failure; keep candidate as-is.
- Add lightweight in-memory URL->thumbnail cache per match invocation to prevent duplicate fetches.

### Matching task updates

In `pkg/tasks/pmv_match.go`:

- After `SearchPMVHaven`, enrich each candidate thumbnail (phase 1).
- Prefer enriched thumbnail in candidate data.
- Keep rank/autolink logic unchanged for first iteration.

Optional later improvement:

- Add small confidence bonus when thumbnail is present and scene URL is valid.

### Batch task

In `pkg/tasks/pmv_match.go` (or new file):

- `func MatchPMVUnmatchedFiles(req MatchPMVBatchRequest) (*MatchPMVBatchResult, int, error)`
- Iterate files and call existing `MatchPMVFile` per file (shared logic).
- Protect API latency:
- support async mode later if needed.
- initial version can be synchronous with `limit`.

In `pkg/api/tasks.go`:

- Add request/response structs and route for `/pmv-match-unmatched`.

## Data/DB Impact

- No schema migration required.
- Uses existing `scenes.cover_url`, `scenes.images`, `files.scene_id`, `scenes.filenames_arr`.

## Error Handling Requirements

- Network/search failures return per-file error in batch mode and continue.
- 404/empty PMVHaven result is non-fatal: mark as `no candidates`.
- Thumbnail parse failure is non-fatal if matching candidate exists.
- Dry-run must never modify DB.

## Logging

Add structured logs with `task=pmv-match` and `task=pmvhaven-scraper`:

- query text
- candidate count
- enrichment fetch URL/status
- chosen thumbnail source (`search_html` vs `scene_html`)
- autolink decision and confidence

## Test Plan

### Unit tests (`pkg/scrape/pmvhaven_test.go`)

- Parse thumbnail from:
- `og:image`
- `twitter:image`
- JSON-LD
- `video poster`
- Prefer non-empty first valid URL and canonicalize relative URLs.

### Unit tests (`pkg/tasks/pmv_match_test.go`)

- Candidate enrichment keeps existing thumbnail if scene fetch fails.
- Candidate enrichment replaces missing thumbnail.
- Batch request filters and counters.
- Dry-run batch does not persist links.

### Integration checks

- Call `POST /api/task/pmv-match` with known `file_id` and verify:
- `matched_scene_id` set when confidence >= threshold
- scene `cover_url` is populated from enriched thumbnail

## Non-goals (for this iteration)

- Full browser automation / JS-rendered scraping.
- Cross-site fallback beyond PMVHaven.
- Replacing existing rescan/matching architecture.
- UI workflow changes (can be added later once API stabilizes).

## Rollout Plan

1. Implement phase 1 enrichment + tests.
2. Validate on a sample set of known PMV filenames.
3. Implement phase 2 batch endpoint + tests.
4. Add UI trigger later (optional) for batch PMV match.

## Open Questions

1. Should batch matching run synchronously (API waits) or as background task with progress events?
2. Should OpenAI reranking remain enabled by default when key is set, or be explicitly opt-in for PMV matching?
3. Should auto-link threshold (`0.85`) be configurable in settings/env?
