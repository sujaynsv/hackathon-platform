---
id: V-002
title: Voting Results — Public Leaderboard
epic: voting
owner: Keerthika (backend)
status: "[ ] not-started"
branch: story/V-002-results
blocks: ADM-003
blocked-by: V-001, J-004
---

# V-002 · Voting Results — Public Leaderboard

## Context (Read ALL of these before writing any code)
- `.agents/rules/engineering-standards.md` — use raw SQL with window functions for ranking; no N+1
- `MASTER-CONTEXT.md` — vote counts cached in Redis for performance; leaderboard only visible after voting closes
- `docs/api-design.md §Voting` — GET /events/{slug}/votes/results
- `docs/data-model.md §votes, §submissions` — `COUNT(*)` from votes table grouped by submission
- `docs/invariants.md` — no specific invariant, but privacy: never expose ip_hash in response

## What We're Building

A leaderboard showing submissions ranked by vote count, available once voting closes. Vote counts are aggregated with a single SQL query using a window function for ranking. Results are cached in Redis for 60 seconds to handle traffic spikes. The leaderboard includes the submission's judging rank alongside its vote rank to show both dimensions.

**Endpoint:** `GET /api/v1/events/{slug}/votes/results`  
**Auth:** Public (but voting results only visible after `voting_closes_at`)

**Query params:** `trackId` (optional filter), `page` / `pageSize`

**Response (200):**
```json
{
  "data": [
    {
      "submissionId": "uuid",
      "title": "AI Hackathon Assistant",
      "team": { "id": "uuid", "name": "Team Rocket" },
      "coverUrl": "...",
      "voteCount": 142,
      "voteRank": 1,
      "judgeScore": 8.47,
      "judgeRank": 2,
      "trackId": "uuid",
      "trackName": "AI Track"
    }
  ],
  "meta": { "page": 1, "totalCount": 45, ... }
}
```
**Error:** Voting hasn't closed yet → 422 `INVARIANT_VIOLATION` with message "results not available until voting closes"

## Files to Create

### internal/voting/port/in.go (add)
```go
type GetResultsUseCase interface {
    GetResults(ctx context.Context, q ResultsQuery) (*ResultsListDTO, error)
}

type ResultsQuery struct {
    EventSlug string
    TrackID   *uuid.UUID
    Page      int
    PageSize  int
}

type ResultsListDTO struct {
    Items      []*VoteResultItem
    TotalCount int
    Page       int
    PageSize   int
}

type VoteResultItem struct {
    SubmissionID string  `json:"submissionId"`
    Title        string  `json:"title"`
    Team         TeamRef `json:"team"`
    CoverURL     *string `json:"coverUrl"`
    VoteCount    int     `json:"voteCount"`
    VoteRank     int     `json:"voteRank"`
    JudgeScore   *float64 `json:"judgeScore"`
    JudgeRank    *int     `json:"judgeRank"`
    TrackID      *string `json:"trackId"`
    TrackName    *string `json:"trackName"`
}

type TeamRef struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}
```

### internal/voting/port/out.go (add)
```go
type ResultsCache interface {
    GetCachedResults(ctx context.Context, key string) ([]byte, error) // nil if miss
    SetCachedResults(ctx context.Context, key string, data []byte, ttl time.Duration) error
}
```

### The Results SQL (single query — no N+1)
```sql
SELECT
    s.id            AS submission_id,
    s.title,
    t.id            AS team_id,
    t.name          AS team_name,
    s.cover_url,
    COUNT(v.id)     AS vote_count,
    RANK() OVER (ORDER BY COUNT(v.id) DESC) AS vote_rank,
    s.final_score   AS judge_score,
    s.overall_rank  AS judge_rank,
    tr.id           AS track_id,
    tr.name         AS track_name,
    COUNT(*) OVER() AS total_count
FROM submissions s
LEFT JOIN votes v    ON v.submission_id = s.id
LEFT JOIN teams t    ON t.id = s.team_id
LEFT JOIN tracks tr  ON tr.id = s.track_id
WHERE s.event_id = $1
  AND s.status = 'submitted'
  [AND s.track_id = $trackId]
GROUP BY s.id, t.id, t.name, tr.id, tr.name
ORDER BY vote_count DESC
LIMIT $2 OFFSET $3;
```

### internal/voting/usecase/results.go
```go
func (s *GetResultsService) GetResults(ctx context.Context, q port.ResultsQuery) (*port.ResultsListDTO, error) {
    // 1. Load event voting info
    event, err := s.events.FindVotingInfoBySlug(ctx, q.EventSlug)
    if err != nil { return nil, err }

    // 2. Check voting has closed (results not available until after closing)
    now := time.Now().UTC()
    if event.VotingClosesAt == nil || now.Before(*event.VotingClosesAt) {
        return nil, fmt.Errorf("%w: results not available until voting closes", response.ErrInvariantViolated)
    }

    // 3. Check Redis cache first
    cacheKey := fmt.Sprintf("results:%s:p%d:ps%d", event.ID, q.Page, q.PageSize)
    if q.TrackID != nil {
        cacheKey += ":" + q.TrackID.String()
    }
    if cached, _ := s.cache.GetCachedResults(ctx, cacheKey); cached != nil {
        var dto port.ResultsListDTO
        if err := json.Unmarshal(cached, &dto); err == nil {
            return &dto, nil
        }
    }

    // 4. Run the results query
    items, total, err := s.votes.GetResults(ctx, event.ID, q.TrackID, q.Page, q.PageSize)
    if err != nil { return nil, err }

    result := &port.ResultsListDTO{Items: items, TotalCount: total, Page: q.Page, PageSize: q.PageSize}

    // 5. Cache for 60 seconds
    if data, err := json.Marshal(result); err == nil {
        _ = s.cache.SetCachedResults(ctx, cacheKey, data, 60*time.Second)
    }

    return result, nil
}
```

## Tests Required

### Use case tests
- `TestGetResultsService_VotingStillOpen_Returns422`
- `TestGetResultsService_AfterVotingCloses_ReturnsRankedList`
- `TestGetResultsService_CacheHit_DoesNotQueryDB` — mock VoteRepository not called when cache returns
- `TestGetResultsService_CacheMiss_QueriesDB_ThenCaches`
- `TestGetResultsService_FilterByTrackID_OnlyReturnsThatTrack`

### Integration test
- Submit votes on 3 submissions → close voting → GET results → verify voteCount and voteRank correct
- Verify `ip_hash` NEVER appears in response
- Verify second call within 60s served from Redis cache

## Definition of Done
- [ ] `go test ./internal/voting/...` → 100% green
- [ ] `GET /events/{slug}/votes/results` before voting closes → 422
- [ ] After voting closes → 200, ranked by vote count descending
- [ ] Results include `judgeScore` and `judgeRank` from normalization (if done)
- [ ] `ip_hash` never exposed in any response field
- [ ] Redis cache: second call within 60s served from cache (no DB query)
- [ ] Track filter works correctly
- [ ] Single SQL query (no N+1 verified by integration test query counter)
