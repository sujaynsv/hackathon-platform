---
id: J-004
title: Score Normalization + Ranking Computation
epic: judging
owner: Keerthika (backend)
status: "[ ] not-started"
branch: story/J-004-normalize
blocks: V-001, V-002, ADM-003
blocked-by: J-002
---

# J-004 · Score Normalization + Ranking Computation

## Context (Read ALL of these before writing any code)
- `.agents/rules/engineering-standards.md` — Go hexagonal rules; use raw SQL CTEs for normalization queries (never ORM)
- `MASTER-CONTEXT.md` — Invariant **I17**: normalization trigger must be audit-logged; normalizationStatus state machine on events table
- `docs/api-design.md §Judging` — POST /events/{slug}/judging/normalize, GET /events/{slug}/judging/results
- `docs/data-model.md §scores (normalized_score), §submissions (final_score, overall_rank, track_rank)` — exact column names
- `docs/invariants.md §I17`

## What We're Building

After all judges have scored (or the deadline passes), an organizer triggers score normalization. This is a compute-heavy step that:
1. Z-score normalizes each judge's raw scores within their own score range (removes judge bias — a harsh judge's 7 and a lenient judge's 9 become comparable)
2. Aggregates normalized scores per submission using the rubric weights
3. Ranks submissions overall and within their track
4. Updates `submissions.final_score`, `submissions.overall_rank`, `submissions.track_rank`
5. Sets `events.normalization_status` to `normalized`

This entire computation runs as a **single PostgreSQL CTE chain** for correctness and performance — never application-side aggregation loops.

**Endpoints:**
- `POST /api/v1/events/{slug}/judging/normalize` — organizer-only: trigger normalization
- `GET /api/v1/events/{slug}/judging/results` — public: ranked results after normalization

### POST /events/{slug}/judging/normalize
**Auth:** Organizer only  
**Response (200):**
```json
{
  "data": {
    "eventId": "uuid",
    "normalizationStatus": "normalized",
    "submissionsRanked": 42,
    "triggeredAt": "..."
  },
  "meta": {...}
}
```
**Error cases:**
| Condition | HTTP | Code |
|-----------|------|------|
| Not organizer | 403 | `FORBIDDEN` |
| Not all judges scored (some assignments still `pending`) | 422 | `INVARIANT_VIOLATION` |
| Event not in `judging` status | 422 | `INVARIANT_VIOLATION` |

### GET /events/{slug}/judging/results
**Response (200):** Ranked submissions list (same shape as gallery but includes `finalScore`, `overallRank`, `trackRank`)

## The Normalization SQL (CTE Chain)

Run this entire block in a single database transaction:

```sql
-- Step 1: Z-score normalize each judge's scores per criterion
WITH judge_stats AS (
    SELECT
        sc.judge_id,
        sc.criterion_id,
        AVG(sc.raw_score)    AS mean_score,
        STDDEV(sc.raw_score) AS stddev_score
    FROM scores sc
    JOIN judge_assignments ja ON ja.id = sc.assignment_id
    WHERE ja.event_id = $1
    GROUP BY sc.judge_id, sc.criterion_id
),
normalized AS (
    SELECT
        sc.id,
        sc.assignment_id,
        sc.criterion_id,
        sc.judge_id,
        CASE
            WHEN js.stddev_score = 0 OR js.stddev_score IS NULL THEN sc.raw_score::NUMERIC
            ELSE (sc.raw_score - js.mean_score) / js.stddev_score
        END AS z_score
    FROM scores sc
    JOIN judge_scores_per_judge js
      ON js.judge_id = sc.judge_id AND js.criterion_id = sc.criterion_id
),
-- Step 2: Weighted aggregate per submission
submission_scores AS (
    SELECT
        ja.submission_id,
        SUM(n.z_score * rc.weight) AS weighted_score
    FROM normalized n
    JOIN judge_assignments ja ON ja.id = n.assignment_id
    JOIN rubric_criteria rc ON rc.id = n.criterion_id
    WHERE ja.status = 'completed'
    GROUP BY ja.submission_id
),
-- Step 3: Overall rank
overall_ranked AS (
    SELECT
        submission_id,
        weighted_score,
        RANK() OVER (ORDER BY weighted_score DESC) AS overall_rank
    FROM submission_scores
),
-- Step 4: Track rank
track_ranked AS (
    SELECT
        s.id AS submission_id,
        RANK() OVER (PARTITION BY s.track_id ORDER BY ss.weighted_score DESC) AS track_rank
    FROM submissions s
    JOIN submission_scores ss ON ss.submission_id = s.id
    WHERE s.track_id IS NOT NULL
),
-- Step 5: Write normalized_score back to scores table
update_scores AS (
    UPDATE scores SET normalized_score = n.z_score
    FROM normalized n WHERE scores.id = n.id
    RETURNING scores.id
)
-- Step 6: Write final_score, overall_rank, track_rank to submissions
UPDATE submissions
SET
    final_score   = r.weighted_score,
    overall_rank  = r.overall_rank,
    track_rank    = COALESCE(tr.track_rank, NULL)
FROM overall_ranked r
LEFT JOIN track_ranked tr ON tr.submission_id = r.submission_id
WHERE submissions.id = r.submission_id
  AND submissions.event_id = $1;
```

After running this CTE, update `events.normalization_status = 'normalized'`.

## Files to Create

### internal/judging/port/in.go (add)
```go
type NormalizeScoresUseCase interface {
    Normalize(ctx context.Context, cmd NormalizeCommand) (*NormalizeResultDTO, error)
}
type GetResultsUseCase interface {
    GetResults(ctx context.Context, eventSlug string) ([]*RankedSubmissionDTO, error)
}

type NormalizeCommand struct {
    CallerID  uuid.UUID
    EventSlug string
}
type NormalizeResultDTO struct {
    EventID             string `json:"eventId"`
    NormalizationStatus string `json:"normalizationStatus"`
    SubmissionsRanked   int    `json:"submissionsRanked"`
    TriggeredAt         string `json:"triggeredAt"`
}
```

### internal/judging/usecase/normalize.go
```go
func (s *NormalizeService) Normalize(ctx context.Context, cmd port.NormalizeCommand) (*port.NormalizeResultDTO, error) {
    // 1. Load event, verify organizer role
    event, _ := s.events.FindSummaryBySlug(ctx, cmd.EventSlug)
    role, _ := s.roles.GetRole(ctx, cmd.CallerID, event.ID)
    if role != "organizer" {
        return nil, fmt.Errorf("%w: only organizers can trigger normalization", response.ErrForbidden)
    }

    // 2. Verify event is in judging status
    if event.Status != "judging" {
        return nil, fmt.Errorf("%w: event must be in 'judging' status", response.ErrInvariantViolated)
    }

    // 3. Verify no pending assignments (all judges must have scored or recused)
    pendingCount, _ := s.assignments.CountPending(ctx, event.ID)
    if pendingCount > 0 {
        return nil, fmt.Errorf("%w: %d assignments still pending", response.ErrInvariantViolated, pendingCount)
    }

    // 4. Run the normalization CTE in a single transaction
    count, err := s.normalizer.RunNormalization(ctx, event.ID)
    if err != nil {
        return nil, fmt.Errorf("normalization failed: %w", err)
    }

    // 5. Update normalization_status on event
    if err := s.events.SetNormalizationStatus(ctx, event.ID, "normalized"); err != nil {
        return nil, err
    }

    // 6. I17: audit log
    _ = s.auditLog.Write(ctx, &port.AuditEntry{
        ActorID: cmd.CallerID, Action: "NORMALIZE_SCORES",
        ResourceType: "event", ResourceID: event.ID,
        Changes: map[string]any{"submissions_ranked": count},
    })

    return &port.NormalizeResultDTO{
        EventID: event.ID.String(), NormalizationStatus: "normalized",
        SubmissionsRanked: count, TriggeredAt: time.Now().UTC().Format(time.RFC3339),
    }, nil
}
```

## Tests Required

### Domain tests (no DB)
- `TestNormalization_ZScoreCalc_ZeroStddev_FallsBackToRawScore`
- `TestNormalization_WeightedAggregate_ThreeCriteria_CorrectTotal`

### Use case tests
- `TestNormalizeService_NonOrganizer_Returns403`
- `TestNormalizeService_PendingAssignments_Returns422`
- `TestNormalizeService_ValidTrigger_WritesAuditLog` (I17)

### Integration test (testcontainers)
- Full flow: create event → add judges → submit → judge → normalize
- Verify `submissions.final_score` and `overall_rank` updated in DB
- Verify `events.normalization_status = 'normalized'`
- Verify `GET /events/{slug}/judging/results` returns submissions in rank order

## Definition of Done
- [ ] `go test ./internal/judging/...` → 100% green
- [ ] `POST /events/{slug}/judging/normalize` runs CTE in single transaction → 200
- [ ] Non-organizer → 403
- [ ] Pending assignments → 422
- [ ] `submissions.final_score`, `overall_rank`, `track_rank` populated after normalize
- [ ] `GET /events/{slug}/judging/results` returns ranked list (descending score)
- [ ] Audit log written (I17)
- [ ] Normalization is idempotent (safe to run twice)
