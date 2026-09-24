---
id: S-004
title: Submission Gallery + Disqualify — Public Listing + Admin Action
epic: submissions
owner: Keerthika (backend)
status: "[ ] not-started"
branch: story/S-004-gallery-disqualify
blocks: V-001, V-002
blocked-by: S-002
---

# S-004 · Submission Gallery + Disqualify

## Context (Read ALL of these before writing any code)
- `.agents/rules/engineering-standards.md` — Go hexagonal rules; use raw SQL for pagination + JOINs (no N+1)
- `MASTER-CONTEXT.md` — I16: only admins can disqualify; I17: all write actions require audit log
- `docs/api-design.md §Submissions` — GET /events/{slug}/submissions, POST /submissions/{id}/disqualify
- `docs/data-model.md §submissions, §audit_log` — `status = 'disqualified'`, `final_score`, `overall_rank`
- `docs/invariants.md §I16, §I17`

## What We're Building

A public gallery of all `submitted` submissions for an event, paginated and sortable. Results show cover images, team names, scores (if judging is complete), and rankings. Admins can disqualify a submission, which changes its status to `disqualified` and writes an audit log entry (I16 + I17).

**Endpoints:**
- `GET /api/v1/events/{slug}/submissions` — public: list all submitted submissions for an event
- `POST /api/v1/submissions/{id}/disqualify` — admin-only: disqualify a submission

### GET /events/{slug}/submissions
**Query params:** `page` (default 1), `pageSize` (default 20), `trackId` (optional filter), `sort` (`score` | `createdAt`, default `score` once judging done else `createdAt`)  
**Response (200):**
```json
{
  "data": [
    {
      "id": "uuid", "title": "AI Hackathon Assistant",
      "team": { "id": "uuid", "name": "Team Rocket" },
      "track": { "id": "uuid", "name": "AI Track" },
      "coverUrl": "https://...",
      "status": "submitted",
      "finalScore": 8.47,
      "overallRank": 2,
      "trackRank": 1,
      "submittedAt": "..."
    }
  ],
  "meta": { "page": 1, "pageSize": 20, "totalCount": 45, "totalPages": 3, ... }
}
```
**Note:** `finalScore`, `overallRank`, `trackRank` are only populated after normalization (judging module). The gallery only shows `status = 'submitted'` OR `status = 'disqualified'` (not `draft`).

### POST /submissions/{id}/disqualify
**Auth:** Admin only  
**Request:** `{ "reason": "Plagiarism detected" }`  
**Success (200):** `{ "data": { "disqualified": true, "submissionId": "uuid" }, "meta": {...} }`  
**Error:** Non-admin → 403 `FORBIDDEN`

## Files to Create / Modify

### internal/submissions/port/in.go (add)
```go
type ListSubmissionsUseCase interface {
    List(ctx context.Context, q ListSubmissionsQuery) (*SubmissionListDTO, error)
}
type DisqualifyUseCase interface {
    Disqualify(ctx context.Context, cmd DisqualifyCommand) error
}

type ListSubmissionsQuery struct {
    EventSlug string
    TrackID   *uuid.UUID
    SortBy    string // "score" or "createdAt"
    Page      int
    PageSize  int
}
type DisqualifyCommand struct {
    AdminID      uuid.UUID
    SubmissionID uuid.UUID
    Reason       string
}
```

### internal/submissions/port/out.go (add)
```go
type AuditLogWriter interface {
    Write(ctx context.Context, entry *AuditEntry) error
}
type AuditEntry struct {
    ActorID      uuid.UUID
    Action       string
    ResourceType string
    ResourceID   uuid.UUID
    Changes      map[string]any
}
```

### internal/submissions/usecase/gallery.go
```go
// ListSubmissionsService loads submissions with a single paginated SQL query:
// SELECT s.*, t.name AS team_name, tr.name AS track_name, COUNT(*) OVER() AS total_count
// FROM submissions s
// LEFT JOIN teams t ON t.id = s.team_id
// LEFT JOIN tracks tr ON tr.id = s.track_id
// WHERE s.event_id = $1 AND s.status IN ('submitted', 'disqualified')
// [AND s.track_id = $trackId]
// ORDER BY [final_score DESC NULLS LAST | created_at DESC]
// LIMIT $2 OFFSET $3
```

### internal/submissions/usecase/disqualify.go
```go
func (s *DisqualifyService) Disqualify(ctx context.Context, cmd port.DisqualifyCommand) error {
    // 1. Verify caller is admin (I16) — check users.is_admin via UserReader port
    isAdmin, _ := s.users.IsAdmin(ctx, cmd.AdminID)
    if !isAdmin {
        return fmt.Errorf("%w: only admins can disqualify submissions", response.ErrForbidden)
    }

    // 2. Load submission
    sub, err := s.subs.FindByID(ctx, cmd.SubmissionID)
    if err != nil { return err }

    // 3. Disqualify
    sub.Status = domain.StatusDisqualified
    sub.UpdatedAt = time.Now().UTC()
    if err := s.subs.Update(ctx, sub); err != nil { return err }

    // 4. I17: always write audit log
    return s.auditLog.Write(ctx, &port.AuditEntry{
        ActorID:      cmd.AdminID,
        Action:       "DISQUALIFY_SUBMISSION",
        ResourceType: "submission",
        ResourceID:   sub.ID,
        Changes:      map[string]any{"reason": cmd.Reason, "status": "disqualified"},
    })
}
```

## Tests Required

### Use case tests
- `TestListSubmissionsService_OnlyReturnsSubmittedAndDisqualified`
- `TestListSubmissionsService_PaginationWorks` — page 2 returns correct offset
- `TestListSubmissionsService_FilterByTrackID_OnlyReturnsThatTrack`
- `TestDisqualifyService_NonAdmin_Returns403` (I16)
- `TestDisqualifyService_ValidAdmin_SetsDisqualifiedStatus`
- `TestDisqualifyService_ValidAdmin_WritesAuditLog` (I17) — mock auditLog.Write called

### Handler tests
- `TestGalleryHandler_PublicUser_Returns200WithSubmittedOnly`
- `TestGalleryHandler_NoDraftSubmissions_InResponse`
- `TestDisqualifyHandler_NonAdmin_Returns403`
- `TestDisqualifyHandler_ValidAdmin_Returns200`

### Integration test
- Create 3 submissions, submit 2, leave 1 as draft → GET gallery → only 2 in response
- Admin disqualify submission → verify status = 'disqualified' in DB
- Verify audit_log row created with action = 'DISQUALIFY_SUBMISSION'

## Definition of Done
- [ ] `go test ./internal/submissions/...` → 100% green
- [ ] `GET /events/{slug}/submissions` → paginated list, no drafts visible
- [ ] Track filter works correctly
- [ ] Sort by `score` when available (NULLS LAST) — else by `createdAt`
- [ ] `POST /submissions/{id}/disqualify` → 200, status = `disqualified`
- [ ] Non-admin → 403 `FORBIDDEN` (I16)
- [ ] Audit log entry written after disqualify (I17)
- [ ] Gallery uses single SQL query (no N+1) — verified by test query counter
