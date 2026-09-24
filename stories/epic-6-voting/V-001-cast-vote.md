---
id: V-001
title: Cast Vote — POST /api/v1/events/{slug}/votes
epic: voting
owner: Sujay (backend)
status: "[ ] not-started"
branch: story/V-001-cast-vote
blocks: V-002
blocked-by: S-004, J-004
---

# V-001 · Cast Vote — POST /api/v1/events/{slug}/votes

## Context (Read ALL of these before writing any code)
- `.agents/rules/engineering-standards.md` — Go hexagonal rules; Redis rate limiting
- `MASTER-CONTEXT.md` — Invariant **I5**: one vote per user per submission (UNIQUE on voter_id + submission_id); Invariant **I12**: voting only during `voting_opens_at` → `voting_closes_at` window; Invariant **I18**: IP must be SHA-256 hashed before storing — NEVER store raw IP
- `docs/api-design.md §Voting` — POST /events/{slug}/votes, DELETE /events/{slug}/votes/{submissionId}
- `docs/data-model.md §votes` — `UNIQUE(voter_id, submission_id)`, `ip_hash` column
- `docs/invariants.md §I5, §I12, §I18`

## What We're Building

Authenticated users vote on submitted submissions during the voting window (I12). Each user can vote once per submission (I5). The voter's IP address is SHA-256 hashed with a salt before storage — raw IPs are never persisted anywhere (I18). Votes are rate-limited via Redis to prevent abuse. Users can retract their vote.

**Endpoints:**
- `POST /api/v1/events/{slug}/votes` — cast a vote on a submission
- `DELETE /api/v1/events/{slug}/votes/{submissionId}` — retract a vote

### POST /events/{slug}/votes
**Auth:** Authenticated  
**Request:** `{ "submissionId": "uuid" }`  
**Success (201):**
```json
{ "data": { "voted": true, "submissionId": "uuid", "votedAt": "..." }, "meta": {...} }
```
**Error cases:**
| Condition | HTTP | Code |
|-----------|------|------|
| Voting window not open (I12) | 422 | `INVARIANT_VIOLATION` |
| Already voted on this submission (I5) | 409 | `DUPLICATE_RESOURCE` |
| Submission not found / not submitted | 404 | `NOT_FOUND` |
| Rate limit exceeded (5 votes/min per user) | 429 | `RATE_LIMITED` |
| Voting on own team's submission | 422 | `INVARIANT_VIOLATION` |

### DELETE /events/{slug}/votes/{submissionId}
**Success (200):** `{ "data": { "retracted": true }, "meta": {...} }`  
**Error:** Not voted → 404

## Files to Create

### internal/voting/domain/vote.go
```go
package domain

import (
    "crypto/sha256"
    "encoding/hex"
    "errors"
    "fmt"
    "time"
    "github.com/google/uuid"
)

var ErrVotingWindowClosed = errors.New("voting window is not open")
var ErrAlreadyVoted       = errors.New("already voted on this submission")
var ErrSelfVote           = errors.New("cannot vote on your own team's submission")

type Vote struct {
    ID           uuid.UUID
    SubmissionID uuid.UUID
    VoterID      uuid.UUID
    EventID      uuid.UUID
    IPHash       string   // SHA-256 hex — NEVER raw IP
    CreatedAt    time.Time
}

// HashIP creates the SHA-256 hash of the IP address with a salt (I18).
// The salt comes from config (IP_HASH_SALT env var) — prevents rainbow table attacks.
func HashIP(rawIP, salt string) string {
    data := salt + ":" + rawIP
    sum := sha256.Sum256([]byte(data))
    return hex.EncodeToString(sum[:])
}

// CheckVotingWindow validates that voting is currently open (I12).
func CheckVotingWindow(eventStatus string, opensAt, closesAt *time.Time) error {
    if eventStatus != "voting" {
        return fmt.Errorf("%w: event status is '%s'", ErrVotingWindowClosed, eventStatus)
    }
    now := time.Now().UTC()
    if opensAt != nil && now.Before(*opensAt) {
        return fmt.Errorf("%w: voting opens at %s", ErrVotingWindowClosed, opensAt.Format(time.RFC3339))
    }
    if closesAt != nil && now.After(*closesAt) {
        return fmt.Errorf("%w: voting closed at %s", ErrVotingWindowClosed, closesAt.Format(time.RFC3339))
    }
    return nil
}
```

### internal/voting/port/in.go
```go
package port

import (
    "context"
    "github.com/google/uuid"
)

type CastVoteUseCase interface {
    Cast(ctx context.Context, cmd CastVoteCommand) (*VoteDTO, error)
}
type RetractVoteUseCase interface {
    Retract(ctx context.Context, cmd RetractVoteCommand) error
}

type CastVoteCommand struct {
    VoterID      uuid.UUID
    EventSlug    string
    SubmissionID uuid.UUID
    RawIP        string   // raw IP from X-Real-IP header — hashed in use case
}
type RetractVoteCommand struct {
    VoterID      uuid.UUID
    EventSlug    string
    SubmissionID uuid.UUID
}
type VoteDTO struct {
    Voted        bool   `json:"voted"`
    SubmissionID string `json:"submissionId"`
    VotedAt      string `json:"votedAt"`
}
```

### internal/voting/port/out.go
```go
package port

import (
    "context"
    "github.com/dogfood/platform/internal/voting/domain"
    "github.com/google/uuid"
)

type VoteRepository interface {
    Save(ctx context.Context, vote *domain.Vote) error
    FindByVoterAndSubmission(ctx context.Context, voterID, submissionID uuid.UUID) (*domain.Vote, error)
    Delete(ctx context.Context, voterID, submissionID uuid.UUID) error
    CountBySubmission(ctx context.Context, submissionID uuid.UUID) (int, error)
}

type RateLimiter interface {
    // Check and increment rate limit. Returns error if limit exceeded.
    // Key: "vote_ratelimit:{userID}" — Redis INCR + EXPIRE
    CheckAndIncrement(ctx context.Context, key string, limit int, windowSeconds int) error
}

type EventVotingReader interface {
    FindVotingInfoBySlug(ctx context.Context, slug string) (*VotingEventInfo, error)
}

type VotingEventInfo struct {
    ID           uuid.UUID
    Status       string
    VotingOpensAt  *time.Time
    VotingClosesAt *time.Time
}

type SubmissionVotingReader interface {
    // Returns teamID — needed to prevent self-voting
    FindSubmissionVotingInfo(ctx context.Context, id uuid.UUID) (*SubmissionVotingInfo, error)
}
type SubmissionVotingInfo struct {
    ID     uuid.UUID
    TeamID uuid.UUID
    Status string
}

type TeamMemberReader interface {
    // IsTeamMember checks if voterID is on this team in this event
    IsTeamMember(ctx context.Context, userID, teamID uuid.UUID) (bool, error)
}
```

### internal/voting/usecase/cast.go
```go
func (s *CastVoteService) Cast(ctx context.Context, cmd port.CastVoteCommand) (*port.VoteDTO, error) {
    // 1. Rate limit check (Redis) — 5 votes per 60 seconds per user (I anti-abuse)
    rateKey := fmt.Sprintf("vote_ratelimit:%s", cmd.VoterID)
    if err := s.rateLimiter.CheckAndIncrement(ctx, rateKey, 5, 60); err != nil {
        return nil, fmt.Errorf("%w", response.ErrRateLimited)
    }

    // 2. Load event, check voting window (I12)
    event, err := s.events.FindVotingInfoBySlug(ctx, cmd.EventSlug)
    if err != nil { return nil, err }
    if err := domain.CheckVotingWindow(event.Status, event.VotingOpensAt, event.VotingClosesAt); err != nil {
        return nil, fmt.Errorf("%w: %s", response.ErrInvariantViolated, err.Error())
    }

    // 3. Load submission
    sub, err := s.submissions.FindSubmissionVotingInfo(ctx, cmd.SubmissionID)
    if err != nil || sub.Status != "submitted" {
        return nil, fmt.Errorf("%w: submission not found", response.ErrNotFound)
    }

    // 4. Prevent self-voting (cannot vote for your own team)
    isMember, _ := s.teamMembers.IsTeamMember(ctx, cmd.VoterID, sub.TeamID)
    if isMember {
        return nil, fmt.Errorf("%w: cannot vote for your own team's submission", response.ErrInvariantViolated)
    }

    // 5. I18: hash the IP — NEVER store raw IP
    ipHash := domain.HashIP(cmd.RawIP, s.ipHashSalt)

    // 6. Create and save vote
    now := time.Now().UTC()
    vote := &domain.Vote{
        ID:           uuid.New(),
        SubmissionID: cmd.SubmissionID,
        VoterID:      cmd.VoterID,
        EventID:      event.ID,
        IPHash:       ipHash,
        CreatedAt:    now,
    }
    if err := s.votes.Save(ctx, vote); err != nil {
        // I5: DB UNIQUE constraint catches duplicate vote → 409
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) && pgErr.Code == "23505" {
            return nil, fmt.Errorf("%w: already voted on this submission", response.ErrInvariantViolated)
        }
        return nil, err
    }

    return &port.VoteDTO{Voted: true, SubmissionID: cmd.SubmissionID.String(), VotedAt: now.Format(time.RFC3339)}, nil
}
```

### internal/shared/cache/rate_limiter.go
```go
// Redis-based rate limiter using INCR + EXPIRE
func (c *RedisCache) CheckAndIncrement(ctx context.Context, key string, limit, windowSec int) error {
    pipe := c.client.Pipeline()
    incr := pipe.Incr(ctx, key)
    pipe.Expire(ctx, key, time.Duration(windowSec)*time.Second)
    if _, err := pipe.Exec(ctx); err != nil {
        return err
    }
    if incr.Val() > int64(limit) {
        return fmt.Errorf("rate limit exceeded")
    }
    return nil
}
```

## Tests Required

### Domain tests
- `TestHashIP_DifferentSalts_ProduceDifferentHashes` (I18)
- `TestHashIP_SameIPSameSalt_ProduceSameHash` (deterministic)
- `TestHashIP_NeverContainsRawIP` (verify output doesn't contain rawIP)
- `TestCheckVotingWindow_VotingStatus_WithinWindow_Nil`
- `TestCheckVotingWindow_RegistrationStatus_ReturnsErr` (I12)
- `TestCheckVotingWindow_AfterClosesAt_ReturnsErr` (I12)

### Use case tests
- `TestCastVoteService_ValidVote_Returns201`
- `TestCastVoteService_DuplicateVote_Returns409` (I5)
- `TestCastVoteService_VotingWindowClosed_Returns422` (I12)
- `TestCastVoteService_SelfVote_Returns422` (own team)
- `TestCastVoteService_RateLimitExceeded_Returns429`
- `TestCastVoteService_IPStoredAsHash_NotRaw` (I18) — verify vote.IPHash != rawIP
- `TestRetractVoteService_NotVoted_Returns404`
- `TestRetractVoteService_ValidRetract_Returns200`

### Integration test
- Vote on submission → 201 → vote again → 409 (I5)
- Verify `votes.ip_hash` is SHA-256 hex, not the test IP address
- Vote after window closes → 422 (I12)

## Definition of Done
- [ ] `go test ./internal/voting/...` → 100% green
- [ ] `POST /events/{slug}/votes` → 201 with `voted: true`
- [ ] Duplicate vote → 409 (I5)
- [ ] Window closed → 422 (I12)
- [ ] Rate limit (> 5/min) → 429 `RATE_LIMITED`
- [ ] `ip_hash` in DB is NEVER equal to raw IP — always SHA-256 (I18)
- [ ] Self-vote (own team) → 422
- [ ] `DELETE /events/{slug}/votes/{submissionId}` → 200 retract
