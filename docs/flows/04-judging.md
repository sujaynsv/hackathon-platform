# Flow 4 — Judging

> Covers the judge evaluation workflow from queue view through score submission.
> **Invariants enforced**: I6, I9, I10

---

## Flow Diagram

```mermaid
sequenceDiagram
    actor J as Judge
    participant Auth as Auth Middleware
    participant Assign as Assignment Service
    participant Score as Score Service
    participant DB as Database
    participant Audit as Audit Log

    %% View queue
    J->>Auth: GET /judge/assignments (JWT in header)
    Auth->>Auth: Verify JWT, load user
    Auth->>Auth: Check: user has judge role for event? (I10)
    Auth->>Assign: Route to assignment service
    Assign->>DB: SELECT assignments WHERE judge_id = me AND event_id = event
    Assign->>DB: JOIN submissions, team details
    Assign-->>J: List of assignments with status
    note over J: Sees: pending, in_progress, completed

    %% Open a submission
    J->>Assign: GET /judge/assignments/{id}
    Assign->>Assign: Check: assignment belongs to this judge?
    Assign->>DB: SELECT submission details (title, description, demo_url, etc.)
    Assign->>DB: SELECT rubric criteria for this track
    Assign->>DB: SELECT existing scores WHERE judge_id = me, assignment_id = id
    note over Assign: Does NOT query other judges' scores (I9)
    Assign-->>J: Submission details + rubric criteria + my existing scores

    %% Update assignment status to in_progress
    Assign->>DB: UPDATE assignment SET status=in_progress (if was pending)

    %% Judge scores a criterion (save partial)
    J->>Score: POST /judge/assignments/{id}/scores { criterion_id, raw_score, comment }
    Score->>Score: Check: now < event.judging_deadline_at? (I6)
    Score->>Score: Check: raw_score <= criterion.max_score?
    Score->>Score: Check: raw_score >= 0?
    Score->>DB: UPSERT score (assignment_id, criterion_id)
    Score->>Audit: WRITE score.created or score.updated
    Score-->>J: 200 OK { score saved }

    note over J: Judge can save multiple times (partial scoring is allowed)

    %% Judge submits complete evaluation
    J->>Score: POST /judge/assignments/{id}/complete
    Score->>Score: Check: now < event.judging_deadline_at? (I6)
    Score->>Score: Check: all criteria have scores?
    Score->>DB: UPDATE assignment SET status=completed, completed_at=now()
    Score->>Audit: WRITE judge.assignment_completed
    Score-->>J: 200 OK { completed_at }
```

---

## Judge's Assignment Queue View

```
My Assignments — Dogfood 2026

[●] Project Alpha         — in_progress  (3 of 4 criteria scored)
[○] ByteBuilder           — pending
[○] HackMate              — pending
[✓] Neural Kitchen        — completed
[✓] OpenRuby              — completed
[!] CollabCanvas          — recused

Progress: 2 of 5 active assignments completed
```

---

## Score Entry UI Logic

For each criterion in the rubric, the judge sees:

```
Criterion: Technical Complexity (weight: 30%, max: 10)
Description: Evaluate the depth and quality of the technical implementation...

Score: [  7  ] / 10
Comment (optional): [The architecture is clean but the API lacks input validation...]

[Save]
```

Scoring rules enforced client-side AND server-side (I6):
- `raw_score` must be between `0` and `criterion.max_score`
- `raw_score` must be a number (decimal allowed)
- Scores can be updated until `judging_deadline_at` (I6)
- After deadline: all score inputs are disabled (read-only UI + server rejection)

---

## Score Isolation (I9)

During the judging window, the Score query always includes:

```sql
SELECT s.*
FROM scores s
WHERE s.assignment_id = $assignment_id
  AND s.judge_id = $current_user_id  -- ALWAYS filtered to current judge
```

The submission detail page for a judge:
- Shows the submission content (title, description, demo_url, repo_url, video_url)
- Shows the rubric criteria
- Shows **only the current judge's own scores**
- Does **NOT** show: number of judges assigned, other judges' scores, overall score

---

## Role Isolation (I10)

```mermaid
flowchart TD
    A["Incoming request to /judge/*"]
    B["Auth middleware: verify JWT"]
    C["Load user's event role"]
    D{"Role = 'judge'\nfor this event?"}
    E["403 Forbidden\n'Judging access requires judge role'"]
    F["Proceed to handler"]

    A --> B
    B --> C
    C --> D
    D -->|No| E
    D -->|Yes| F
```

Note: An organizer who is also invited as a judge has **two separate role records** for the event. The judging endpoint checks for the judge role specifically.

---

## Error Cases

| Scenario | HTTP Status | Error Code |
|----------|-------------|------------|
| No judge role for event (I10) | 403 | `NOT_A_JUDGE` |
| Assignment belongs to different judge | 403 | `ASSIGNMENT_NOT_YOURS` |
| Score after judging deadline (I6) | 403 | `JUDGING_DEADLINE_PASSED` |
| Score out of range | 422 | `SCORE_OUT_OF_RANGE` |
| Complete with missing criteria | 422 | `INCOMPLETE_EVALUATION` |
