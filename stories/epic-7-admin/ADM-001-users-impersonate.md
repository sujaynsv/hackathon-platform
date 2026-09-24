---
id: ADM-001
title: User Management + Impersonation — Admin Endpoints
epic: admin
owner: Sujay (backend)
status: "[ ] not-started"
branch: story/ADM-001-users-impersonate
blocks: ADM-002
blocked-by: A-002
---

# ADM-001 · User Management + Impersonation

## Context (Read ALL of these before writing any code)
- `.agents/rules/engineering-standards.md` — Go hexagonal rules
- `MASTER-CONTEXT.md` — Invariant **I17**: ALL admin write actions must be audit-logged; Invariant **I18**: impersonation starts an audit log entry BEFORE impersonation begins (not after)
- `docs/api-design.md §Admin` — GET /admin/users, PATCH /admin/users/{id}, POST /admin/users/{id}/impersonate
- `docs/data-model.md §users, §audit_log` — `is_active`, `is_admin` columns; audit_log append-only
- `docs/invariants.md §I17, §I18`

## What We're Building

Admin-only endpoints to list users, deactivate accounts, promote to admin, and impersonate users (get a JWT that acts as that user). Impersonation is a powerful action — the audit log entry is written BEFORE issuing the impersonation token (I18), so there's no way to impersonate without a log trail, even if something crashes after logging.

**Endpoints:**
- `GET /api/v1/admin/users` — paginated user list with filters
- `PATCH /api/v1/admin/users/{id}` — update user (activate/deactivate, promote admin)
- `POST /api/v1/admin/users/{id}/impersonate` — get impersonation token

All endpoints: **admin-only** (is_admin = true on the JWT)

### GET /admin/users
**Query params:** `page`, `pageSize`, `search` (email/name contains), `isActive` (bool filter)  
**Response (200):** Paginated list of UserProfile DTOs + `totalCount`

### PATCH /admin/users/{id}
**Request (all optional):**
```json
{ "isActive": false, "isAdmin": true, "displayName": "New Name" }
```
**Success (200):** Updated UserProfile DTO  
**Guards:** Admin cannot deactivate themselves; cannot remove their own admin status

### POST /admin/users/{id}/impersonate
**Response (200):**
```json
{
  "data": {
    "impersonationToken": "eyJ...",
    "targetUserId": "uuid",
    "targetEmail": "user@example.com",
    "expiresIn": 3600
  },
  "meta": {...}
}
```
**Business rules:**
- Impersonation token is a regular JWT with `sub = targetUserId`, `is_admin = false` (impersonator does NOT get admin rights as the target), and `jti = new UUID`
- TTL: 1 hour (hardcoded — shorter than normal access tokens for safety)
- **I18**: Audit log is written FIRST, before token issuance — if token issuance fails, the log still exists

## Files to Create

### internal/admin/domain/audit.go
```go
package domain

import (
    "time"
    "github.com/google/uuid"
)

// AuditLog is append-only. No Update or Delete methods exist on this struct.
// Database-level REVOKE enforces this (migration 022).
type AuditLog struct {
    ID           uuid.UUID
    ActorID      *uuid.UUID
    Action       string
    ResourceType string
    ResourceID   *uuid.UUID
    Changes      map[string]any
    IPAddress    *string  // SHA-256 hash if provided, never raw
    CreatedAt    time.Time
}
```

### internal/admin/port/in.go
```go
package port

import (
    "context"
    "github.com/google/uuid"
)

type ListUsersUseCase interface {
    ListUsers(ctx context.Context, q UsersQuery) (*UserListDTO, error)
}
type UpdateUserUseCase interface {
    UpdateUser(ctx context.Context, cmd UpdateUserCommand) (*UserProfileDTO, error)
}
type ImpersonateUseCase interface {
    Impersonate(ctx context.Context, cmd ImpersonateCommand) (*ImpersonationDTO, error)
}

type UsersQuery struct {
    Search   string
    IsActive *bool
    Page     int
    PageSize int
}
type UpdateUserCommand struct {
    AdminID     uuid.UUID
    TargetID    uuid.UUID
    IsActive    *bool
    IsAdmin     *bool
    DisplayName *string
}
type ImpersonateCommand struct {
    AdminID  uuid.UUID
    TargetID uuid.UUID
}
type ImpersonationDTO struct {
    ImpersonationToken string `json:"impersonationToken"`
    TargetUserID       string `json:"targetUserId"`
    TargetEmail        string `json:"targetEmail"`
    ExpiresIn          int    `json:"expiresIn"`
}
```

### internal/admin/usecase/impersonate.go
```go
func (s *ImpersonateService) Impersonate(ctx context.Context, cmd port.ImpersonateCommand) (*port.ImpersonationDTO, error) {
    // 1. Verify caller is admin
    admin, err := s.users.FindByID(ctx, cmd.AdminID)
    if err != nil || !admin.IsAdmin {
        return nil, fmt.Errorf("%w: admin access required", response.ErrForbidden)
    }

    // 2. Load target user
    target, err := s.users.FindByID(ctx, cmd.TargetID)
    if err != nil { return nil, err }

    // 3. I18: Write audit log BEFORE issuing token
    // Even if token issuance fails below, the log exists.
    if err := s.auditLog.Write(ctx, &domain.AuditLog{
        ID:           uuid.New(),
        ActorID:      &cmd.AdminID,
        Action:       "IMPERSONATE_USER",
        ResourceType: "user",
        ResourceID:   &cmd.TargetID,
        Changes:      map[string]any{"target_email": target.Email},
        CreatedAt:    time.Now().UTC(),
    }); err != nil {
        return nil, fmt.Errorf("write audit log: %w", err) // fail if audit log fails
    }

    // 4. Issue impersonation token (1-hour TTL, is_admin=false)
    token, err := s.tokens.IssueImpersonationToken(target.ID.String(), target.Email)
    if err != nil { return nil, err }

    return &port.ImpersonationDTO{
        ImpersonationToken: token,
        TargetUserID:       target.ID.String(),
        TargetEmail:        target.Email,
        ExpiresIn:          3600,
    }, nil
}
```

### internal/admin/usecase/update_user.go
```go
func (s *UpdateUserService) UpdateUser(ctx context.Context, cmd port.UpdateUserCommand) (*port.UserProfileDTO, error) {
    // 1. Verify admin
    admin, _ := s.users.FindByID(ctx, cmd.AdminID)
    if !admin.IsAdmin {
        return nil, fmt.Errorf("%w", response.ErrForbidden)
    }

    // 2. Guard: cannot deactivate/demote self
    if cmd.TargetID == cmd.AdminID {
        if cmd.IsActive != nil && !*cmd.IsActive {
            return nil, fmt.Errorf("%w: admin cannot deactivate their own account", response.ErrInvariantViolated)
        }
        if cmd.IsAdmin != nil && !*cmd.IsAdmin {
            return nil, fmt.Errorf("%w: admin cannot remove their own admin status", response.ErrInvariantViolated)
        }
    }

    // 3. Apply updates
    target, err := s.users.FindByID(ctx, cmd.TargetID)
    if err != nil { return nil, err }
    if cmd.IsActive != nil { target.IsActive = *cmd.IsActive }
    if cmd.IsAdmin != nil { target.IsAdmin = *cmd.IsAdmin }
    if cmd.DisplayName != nil { target.DisplayName = *cmd.DisplayName }

    if err := s.users.Update(ctx, target); err != nil { return nil, err }

    // 4. I17: audit log
    _ = s.auditLog.Write(ctx, &domain.AuditLog{
        ActorID:      &cmd.AdminID,
        Action:       "UPDATE_USER",
        ResourceType: "user",
        ResourceID:   &cmd.TargetID,
        Changes:      buildChanges(cmd),
        CreatedAt:    time.Now().UTC(),
    })

    return toUserProfileDTO(target), nil
}
```

## Tests Required

### Use case tests
- `TestImpersonateService_NonAdmin_Returns403`
- `TestImpersonateService_ValidAdmin_WritesAuditLogBeforeToken` (I18) — mock auditLog verify called before tokens.Issue
- `TestImpersonateService_TokenTTL_Is3600Seconds`
- `TestUpdateUserService_AdminDeactivatesSelf_Returns422`
- `TestUpdateUserService_AdminRemovesOwnAdmin_Returns422`
- `TestUpdateUserService_DeactivateOtherUser_Succeeds_WritesAuditLog` (I17)

### Integration test
- Admin impersonates user → verify audit_log row written
- Admin deactivates themselves → 422
- Use impersonation token → acts as target user (not admin)

## Definition of Done
- [ ] `go test ./internal/admin/...` → 100% green
- [ ] `POST /admin/users/{id}/impersonate` → audit log written BEFORE token returned (I18)
- [ ] Non-admin → 403
- [ ] Admin cannot deactivate/demote self → 422
- [ ] `PATCH /admin/users/{id}` changes apply and are audit-logged (I17)
- [ ] `GET /admin/users` → paginated list with search + filter
- [ ] Impersonation token has `is_admin=false` even if target is admin
