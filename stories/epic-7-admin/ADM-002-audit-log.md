---
id: ADM-002
title: Audit Log — Immutable Activity Trail
epic: admin
owner: Keerthika (backend)
status: "[ ] not-started"
branch: story/ADM-002-audit-log
blocks: ADM-003
blocked-by: ADM-001
---

# ADM-002 · Audit Log — Immutable Activity Trail

## Context (Read ALL of these before writing any code)
- `.agents/rules/engineering-standards.md` — Go hexagonal rules
- `MASTER-CONTEXT.md` — Invariant **I7**: `audit_log` table is append-only — `app_user` role has UPDATE and DELETE revoked at the database level (migration 022); Invariant **I17**: every write action across all modules must produce an audit log entry
- `docs/api-design.md §Admin` — GET /admin/audit-log (paginated, filterable)
- `docs/data-model.md §audit_log` — exact columns: actor_id, action, resource_type, resource_id, changes JSONB, ip_address (SHA-256 hashed)
- `docs/invariants.md §I7, §I17`

## What We're Building

A query endpoint for the audit log. Admins can search and filter the complete activity trail. The write path is already being used by all other modules (via `AuditLogWriter` port) — this story implements the read path and verifies the append-only guarantee.

**I7 guarantee:** The Go app uses `app_user` database role. That role cannot UPDATE or DELETE from `audit_log` (revoked in migration 022). So even if a bug in the code tries to delete an audit log entry, the database will reject it with a permission error. The test MUST verify this.

**Endpoint:** `GET /api/v1/admin/audit-log`  
**Auth:** Admin only

**Query params:**
- `actorId` — filter by who performed the action
- `action` — filter by action type (e.g., `IMPERSONATE_USER`, `DISQUALIFY_SUBMISSION`)
- `resourceType` — `user`, `event`, `submission`, `judge_assignment`, etc.
- `resourceId` — filter by specific resource UUID
- `from` / `to` — ISO8601 datetime range
- `page` / `pageSize`

**Response (200):**
```json
{
  "data": [
    {
      "id": "uuid",
      "actorId": "uuid",
      "actorEmail": "admin@example.com",
      "action": "DISQUALIFY_SUBMISSION",
      "resourceType": "submission",
      "resourceId": "uuid",
      "changes": { "reason": "Plagiarism", "status": "disqualified" },
      "createdAt": "2026-10-01T15:30:00Z"
    }
  ],
  "meta": { "page": 1, "totalCount": 1203, ... }
}
```
**Note:** `ip_address` column is NEVER returned in the response — it is stored as a SHA-256 hash and is only used internally for rate-limit forensics.

## Files to Create

### internal/admin/port/in.go (add)
```go
type QueryAuditLogUseCase interface {
    Query(ctx context.Context, q AuditLogQuery) (*AuditLogListDTO, error)
}

type AuditLogQuery struct {
    ActorID      *uuid.UUID
    Action       *string
    ResourceType *string
    ResourceID   *uuid.UUID
    From         *time.Time
    To           *time.Time
    Page         int
    PageSize     int
}

type AuditLogItem struct {
    ID           string         `json:"id"`
    ActorID      *string        `json:"actorId"`
    ActorEmail   *string        `json:"actorEmail"` // JOINed from users table
    Action       string         `json:"action"`
    ResourceType string         `json:"resourceType"`
    ResourceID   *string        `json:"resourceId"`
    Changes      map[string]any `json:"changes"`
    CreatedAt    string         `json:"createdAt"`
    // ip_address INTENTIONALLY OMITTED from DTO
}
```

### internal/admin/port/out.go (add)
```go
type AuditLogRepository interface {
    // Write is called by ALL modules — this is the shared append port.
    // Never expose Update or Delete from this interface.
    Write(ctx context.Context, entry *domain.AuditLog) error
    Query(ctx context.Context, q port.AuditLogQuery) ([]*AuditLogRow, int, error)
}

type AuditLogRow struct {
    ID           uuid.UUID
    ActorID      *uuid.UUID
    ActorEmail   *string
    Action       string
    ResourceType string
    ResourceID   *uuid.UUID
    Changes      []byte // JSONB raw
    CreatedAt    time.Time
    TotalCount   int // window function
}
```

### internal/admin/repository/audit_log.go
```go
// Write: standard INSERT, no UPDATE/DELETE (app_user role will reject those anyway)
func (r *AuditLogRepository) Write(ctx context.Context, entry *domain.AuditLog) error {
    const q = `
        INSERT INTO audit_log (id, actor_id, action, resource_type, resource_id, changes, ip_address, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
    changesJSON, _ := json.Marshal(entry.Changes)
    _, err := r.db.ExecContext(ctx, q,
        entry.ID, entry.ActorID, entry.Action, entry.ResourceType,
        entry.ResourceID, changesJSON, entry.IPAddress, entry.CreatedAt,
    )
    return err
}

// Query: JOIN with users table for actor email, COUNT(*) OVER() for pagination
// SQL:
// SELECT
//   al.*, u.email AS actor_email,
//   COUNT(*) OVER() AS total_count
// FROM audit_log al
// LEFT JOIN users u ON u.id = al.actor_id
// WHERE 1=1
//   [AND al.actor_id = $actorId]
//   [AND al.action = $action]
//   [AND al.resource_type = $resourceType]
//   [AND al.resource_id = $resourceId]
//   [AND al.created_at >= $from]
//   [AND al.created_at <= $to]
// ORDER BY al.created_at DESC
// LIMIT $n OFFSET $m
```

### internal/admin/usecase/audit_log.go
```go
func (s *QueryAuditLogService) Query(ctx context.Context, q port.AuditLogQuery) (*port.AuditLogListDTO, error) {
    // 1. Verify caller is admin (handled by JWTMiddleware + admin check in handler)
    // 2. Query repository
    rows, total, err := s.auditLog.Query(ctx, q)
    if err != nil { return nil, err }
    // 3. Map to DTOs (ip_address is NOT included in mapping)
    items := make([]*port.AuditLogItem, len(rows))
    for i, row := range rows {
        var changes map[string]any
        json.Unmarshal(row.Changes, &changes)
        items[i] = &port.AuditLogItem{
            ID:           row.ID.String(),
            ActorID:      uuidPtrToStr(row.ActorID),
            ActorEmail:   row.ActorEmail,
            Action:       row.Action,
            ResourceType: row.ResourceType,
            ResourceID:   uuidPtrToStr(row.ResourceID),
            Changes:      changes,
            CreatedAt:    row.CreatedAt.Format(time.RFC3339),
            // ip_address: intentionally omitted
        }
    }
    return &port.AuditLogListDTO{Items: items, TotalCount: total, Page: q.Page, PageSize: q.PageSize}, nil
}
```

## Tests Required

### Repository integration tests (testcontainers — MUST use real PostgreSQL)
- `TestAuditLogRepository_Write_InsertsRow_Successfully`
- `TestAuditLogRepository_Write_AppOnly_CannotUpdateAuditLog` (I7) — attempt UPDATE from app_user → verify permission denied error
- `TestAuditLogRepository_Write_AppOnly_CannotDeleteAuditLog` (I7) — attempt DELETE from app_user → verify permission denied error
- `TestAuditLogRepository_Query_ReturnsFilteredByAction`
- `TestAuditLogRepository_Query_ReturnsFilteredByDateRange`
- `TestAuditLogRepository_Query_NeverReturnsIPAddress` — verify no field in response contains raw IP-like value

### Use case tests
- `TestQueryAuditLogService_AdminOnly_ReturnsEntries`
- `TestQueryAuditLogService_IPAddressNeverInDTO` — mock row has ip_address set → DTO must not contain it

## Definition of Done
- [ ] `go test ./internal/admin/...` → 100% green
- [ ] `GET /admin/audit-log` → paginated, filterable audit entries
- [ ] `ip_address` field NEVER appears in API response
- [ ] **I7 test passes**: `app_user` UPDATE on audit_log → PostgreSQL returns permission denied error
- [ ] **I7 test passes**: `app_user` DELETE on audit_log → PostgreSQL returns permission denied error
- [ ] All filters (actorId, action, resourceType, from/to) work correctly
- [ ] Entries from all modules (disqualify, impersonate, recuse, normalize) appear in log
