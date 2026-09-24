# Flow 6 — Community Voting (T3)

> Covers the public voting system with anti-abuse measures.
> **Invariants enforced**: I5, I12
> **Threat surface**: Sybil voting, ballot stuffing, bandwagon effect, rate abuse

---

## Flow Diagram

```mermaid
sequenceDiagram
    actor U as Authenticated User
    participant Auth as Auth Middleware
    participant Vote as Vote Service
    participant RateLimit as Rate Limiter
    participant DB as Database
    participant Audit as Audit Log

    %% View gallery (randomized order)
    U->>Vote: GET /events/{slug}/gallery
    Vote->>Vote: Check: event.status = voting?
    Vote->>DB: SELECT submissions WHERE status='submitted'\nORDER BY gallery_order (pre-randomized per session)
    Vote->>Vote: Scores and vote counts are HIDDEN (I12)
    Vote-->>U: Gallery with randomized project order\n(no scores, no vote counts)

    %% Cast a vote
    U->>Auth: POST /events/{slug}/votes { submission_id }
    Auth->>Auth: Verify JWT (must be logged in)
    Auth->>Vote: Route to vote service

    Vote->>RateLimit: Check rate limit for user_id
    RateLimit->>RateLimit: Count votes in last 1 hour for this user
    alt Rate limit exceeded
        RateLimit-->>U: 429 Too Many Requests\n{ retry_after: 3600 }
    else Within limit
        Vote->>Vote: Check: event.status = voting?
        Vote->>Vote: Check: submission.status = submitted?
        Vote->>Vote: Check: now between voting_opens_at and voting_closes_at?
        Vote->>DB: SELECT 1 FROM votes\nWHERE voter_id = me AND submission_id = sub_id
        
        alt Already voted for this submission (I5)
            Vote-->>U: 409 Conflict\n"Already voted for this project"
        else First vote for this submission
            Vote->>Vote: Hash IP address (SHA-256)
            Vote->>DB: INSERT vote { voter_id, submission_id, ip_hash, voted_at }
            Vote->>Audit: WRITE vote.cast { submission_id, ip_hash }
            Vote-->>U: 201 Created ✓
        end
    end
```

---

## Anti-Abuse Mechanisms

```mermaid
graph TD
    subgraph "Abuse Vectors & Defenses"
        A["🚨 Sybil Voting\n(Many accounts, same person)"]
        B["🚨 Ballot Stuffing\n(Vote for same project repeatedly)"]
        C["🚨 Bandwagon Effect\n(Vote for popular project because it shows high count)"]
        D["🚨 Rate Abuse\n(Automated voting script)"]
        E["🚨 Submission Scraping\n(Mass-register, view all submissions without voting)"]

        A -->|Defense| A1["IP hash logging\nCorrelate accounts with same IP pattern\nOrganizer dashboard flags suspicious accounts"]
        B -->|Defense| B1["DB unique constraint (I5)\n(voter_id, submission_id) unique\nRejected at DB level, not just application"]
        C -->|Defense| C1["Vote counts HIDDEN during window (I12)\nGallery shows projects in randomized order\nOrder randomized per session (different each load)"]
        D -->|Defense| D1["Rate limit: N votes per hour per user\nJWT required (no anonymous voting)\nIP-based rate limit as secondary check"]
        E -->|Defense| E1["Gallery is always accessible (no defense needed)\nVoting requires verified account\nSubmission rate: audit trail only"]
    end
```

---

## Gallery Randomization Strategy

To prevent the bandwagon effect where projects at the top get more votes:

```
On each gallery request:
  1. Submissions are pre-assigned a stable `gallery_order` (random integer) at submission time
  2. Gallery is always sorted by `gallery_order` (consistent within a session)
  3. A "re-shuffle" option allows users to see a different random order
  
Why stable gallery_order?
  - Prevents infinite re-shuffling abuse
  - Still randomizes compared to submission order
  - Consistent experience per load (no flicker)
```

---

## Vote Count Visibility Rules (I12)

| Event Status | Voter sees own vote? | Vote counts visible? | Final results visible? |
|-------------|---------------------|---------------------|----------------------|
| `submissions_open` | ❌ | ❌ | ❌ |
| `judging` | ❌ | ❌ | ❌ |
| `voting` | ✅ (own vote only) | ❌ | ❌ |
| `results_published` | ✅ | ✅ | ✅ |

---

## Rate Limiting Configuration

```
Per-user vote rate limit:
  Window: 1 hour
  Max votes: configurable per event (default: 10 per hour)
  Strategy: sliding window (not fixed window)

Per-IP secondary limit:
  Window: 1 hour  
  Max votes: 3× the per-user limit
  Purpose: Catch coordinated voting from same network
```

---

## Audit Trail for Voting

Every vote is logged to AuditLog with:
```json
{
  "action": "vote.cast",
  "actor_id": "user-uuid",
  "target_type": "submission",
  "target_id": "submission-uuid",
  "metadata": {
    "ip_hash": "sha256-of-ip",
    "event_id": "event-uuid"
  }
}
```

Organizer can export `audit_log` filtered by `action = 'vote.cast'` to detect anomalies.

---

## Sybil Detection (Organizer Dashboard)

The organizer sees flagged accounts:
```
⚠️  Suspicious voting patterns detected:

IP Hash abc123:
  - 14 accounts voted from this IP hash
  - All created within 2 hours
  - All voted for submission "Project X" only

Recommended action: [Review] [Disqualify Votes] [Ignore]
```

---

## Error Cases

| Scenario | HTTP Status | Error Code |
|----------|-------------|------------|
| Not authenticated | 401 | `UNAUTHORIZED` |
| Voting window not open | 422 | `VOTING_NOT_OPEN` |
| Already voted (I5) | 409 | `ALREADY_VOTED` |
| Rate limit exceeded | 429 | `RATE_LIMIT_EXCEEDED` |
| Submission disqualified | 422 | `SUBMISSION_NOT_ELIGIBLE` |
