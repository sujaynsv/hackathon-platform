---
id: ADM-003
title: Certificate Generation + Verification
epic: admin
owner: Sujay (backend)
status: "[ ] not-started"
branch: story/ADM-003-certificates
blocks: none
blocked-by: J-004, V-002, ADM-001
---

# ADM-003 · Certificate Generation + Verification

## Context (Read ALL of these before writing any code)
- `.agents/rules/engineering-standards.md` — Go hexagonal rules; MinIO for PDF storage
- `MASTER-CONTEXT.md` — Invariant **I17**: certificate generation must be audit-logged; MinIO bucket `certificates`
- `docs/api-design.md §Admin` — POST /admin/events/{slug}/certificates/generate, GET /certificates/{hash}/verify
- `docs/data-model.md §certificates` — `verification_hash VARCHAR(64) UNIQUE`, `achievement`, `issued_at`

## What We're Building

After an event's results are published, admins generate certificates for all team members (participants who submitted) and for winners. Each certificate has a unique verification hash (SHA-256 of eventId+userId+achievement+timestamp+secret). Users can verify any certificate's authenticity via a public endpoint using this hash. Certificates are generated as PDFs using Go's standard `text/template` + a PDF library and stored in MinIO.

**Endpoints:**
- `POST /api/v1/admin/events/{slug}/certificates/generate` — admin-only: generate all certificates for an event
- `GET /api/v1/certificates/{hash}/verify` — public: verify a certificate's authenticity

### POST /admin/events/{slug}/certificates/generate
**Auth:** Admin only  
**Response (200):**
```json
{
  "data": {
    "eventId": "uuid",
    "certificatesIssued": 45,
    "generatedAt": "..."
  },
  "meta": {...}
}
```
**Error:** Event must be in `results_published` status → 422 if not  
**Generates certificates for:** All team members of submitted/non-disqualified teams + special "Winner" certificates for top 3 overall and top 1 per track

**Certificate content includes:**
- Participant's name + event name
- Achievement (e.g., "1st Place — AI Track", "Participant")
- Event dates
- Verification hash (printed on the certificate for public verification)

### GET /certificates/{hash}/verify
**Auth:** Public (no login needed — anyone can verify)  
**Response (200):**
```json
{
  "data": {
    "valid": true,
    "participantName": "Alice Smith",
    "eventName": "Dogfood Hackathon 2026",
    "achievement": "1st Place — AI Track",
    "issuedAt": "2026-11-25T10:00:00Z"
  },
  "meta": {...}
}
```
**Error:** Hash not found → 200 with `{ "data": { "valid": false } }` (never 404 — prevents hash enumeration)

## Files to Create

### internal/admin/domain/certificate.go
```go
package domain

import (
    "crypto/sha256"
    "fmt"
    "encoding/hex"
    "time"
    "github.com/google/uuid"
)

type Certificate struct {
    ID               uuid.UUID
    UserID           uuid.UUID
    EventID          uuid.UUID
    TeamID           *uuid.UUID
    Achievement      string   // e.g. "1st Place — AI Track" or "Participant"
    VerificationHash string   // SHA-256 hex
    IssuedAt         time.Time
}

// GenerateVerificationHash creates a deterministic, unique hash for a certificate.
// Using a server secret prevents forging hashes externally.
func GenerateVerificationHash(eventID, userID uuid.UUID, achievement string, issuedAt time.Time, secret string) string {
    data := fmt.Sprintf("%s:%s:%s:%d:%s",
        eventID, userID, achievement, issuedAt.UnixNano(), secret,
    )
    sum := sha256.Sum256([]byte(data))
    return hex.EncodeToString(sum[:])
}
```

### internal/admin/port/in.go (add)
```go
type GenerateCertificatesUseCase interface {
    Generate(ctx context.Context, cmd GenerateCertCommand) (*GenerateResultDTO, error)
}
type VerifyCertificateUseCase interface {
    Verify(ctx context.Context, hash string) (*VerifyResultDTO, error)
}

type GenerateCertCommand struct {
    AdminID   uuid.UUID
    EventSlug string
}
type GenerateResultDTO struct {
    EventID             string `json:"eventId"`
    CertificatesIssued  int    `json:"certificatesIssued"`
    GeneratedAt         string `json:"generatedAt"`
}
type VerifyResultDTO struct {
    Valid           bool    `json:"valid"`
    ParticipantName *string `json:"participantName,omitempty"`
    EventName       *string `json:"eventName,omitempty"`
    Achievement     *string `json:"achievement,omitempty"`
    IssuedAt        *string `json:"issuedAt,omitempty"`
}
```

### internal/admin/usecase/certificates.go
```go
func (s *GenerateCertificatesService) Generate(ctx context.Context, cmd port.GenerateCertCommand) (*port.GenerateResultDTO, error) {
    // 1. Verify admin, load event
    // 2. Check event is in 'results_published' status
    // 3. Load all team_members for submitted non-disqualified submissions in this event
    // 4. Determine achievements:
    //    - overall_rank = 1: "1st Place Overall"
    //    - overall_rank = 2: "2nd Place Overall"
    //    - overall_rank = 3: "3rd Place Overall"
    //    - track_rank = 1: "1st Place — {Track Name}"
    //    - all others: "Participant"
    // 5. For each team member:
    //    a. Generate verification hash (domain.GenerateVerificationHash)
    //    b. Generate PDF bytes (text/template → pdf)
    //    c. Upload PDF to MinIO bucket 'certificates'
    //    d. Save certificate row to DB
    // 6. I17: audit log
    // 7. Return count
}

func (s *VerifyCertificateService) Verify(ctx context.Context, hash string) (*port.VerifyResultDTO, error) {
    cert, err := s.certs.FindByHash(ctx, hash)
    if err != nil || cert == nil {
        // ALWAYS return valid=false, never 404 (prevents hash enumeration)
        return &port.VerifyResultDTO{Valid: false}, nil
    }
    user, _ := s.users.FindByID(ctx, cert.UserID)
    event, _ := s.events.FindByID(ctx, cert.EventID)
    return &port.VerifyResultDTO{
        Valid:           true,
        ParticipantName: &user.DisplayName,
        EventName:       &event.Title,
        Achievement:     &cert.Achievement,
        IssuedAt:        ptr(cert.IssuedAt.Format(time.RFC3339)),
    }, nil
}
```

### PDF generation
Use `github.com/jung-kurt/gofpdf` or `github.com/signintech/gopdf`:
- Simple A4 template with event name, participant name, achievement, verification hash, date
- Store as `certificates/{eventID}/{userID}.pdf` in MinIO

## Tests Required

### Domain tests
- `TestGenerateVerificationHash_Deterministic_SameInputSameHash`
- `TestGenerateVerificationHash_DifferentUserID_DifferentHash`
- `TestGenerateVerificationHash_WithSecret_CannotBeForgered` — without secret, hash doesn't match

### Use case tests
- `TestGenerateCertificatesService_EventNotResultsPublished_Returns422`
- `TestGenerateCertificatesService_ValidEvent_GeneratesCertsForAllMembers`
- `TestGenerateCertificatesService_WritesAuditLog` (I17)
- `TestVerifyCertificateService_ValidHash_ReturnsValidTrue`
- `TestVerifyCertificateService_InvalidHash_ReturnsValidFalse_Not404` (never 404)

### Integration test
- Generate certs → verify with correct hash → `valid: true`
- Verify with wrong hash → `valid: false` (200, not 404)
- Verify `certificates` table populated with correct achievement labels

## Definition of Done
- [ ] `go test ./internal/admin/...` → 100% green
- [ ] `POST /admin/events/{slug}/certificates/generate` → 200, all team members get certs
- [ ] Top 3 overall get "1st/2nd/3rd Place" achievement
- [ ] Track winners get "1st Place — {Track Name}" achievement
- [ ] `GET /certificates/{hash}/verify` with valid hash → 200 `valid: true`
- [ ] Invalid hash → 200 `valid: false` (never 404)
- [ ] Audit log entry written per generation run (I17)
- [ ] PDF stored in MinIO `certificates` bucket
- [ ] Verification hash is cryptographically sound (uses server secret)
