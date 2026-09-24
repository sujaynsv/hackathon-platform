# Flow 3 — Judge Invitation & Assignment

> Covers inviting judges, configuring assignments, and the algorithmic auto-assign process.
> **Invariants enforced**: I3, I13

---

## Flow Diagram — Judge Invitation

```mermaid
sequenceDiagram
    actor O as Organizer
    participant Judge as Judge Service
    participant Auth as Auth Service
    participant Email as Email (in-app)
    participant DB as Database
    participant Audit as Audit Log

    O->>Judge: POST /events/{id}/judges/invite { email }
    Judge->>DB: Check: user exists with this email?
    
    alt User exists
        Judge->>DB: INSERT event_role (user_id, event_id, role='judge')
        Judge->>Email: Send "You've been invited as a judge" notification
    else User does not exist
        Judge->>DB: INSERT judge_invitation { email, token, event_id }
        Judge->>Email: Send "Create account to judge" link with token
    end

    Judge->>Audit: WRITE judge.invited
    Judge-->>O: 201 Created

    note over O,Email: Judge receives email

    actor J as Judge (new user)
    J->>Auth: POST /auth/register?invite_token=... { display_name, password }
    Auth->>DB: INSERT user + assign judge role for event
    Auth-->>J: 200 OK + JWT issued
```

---

## Flow Diagram — Manual Judge Assignment

```mermaid
sequenceDiagram
    actor O as Organizer
    participant Assign as Assignment Service
    participant DB as Database
    participant Audit as Audit Log

    O->>Assign: POST /events/{id}/assignments { judge_id, submission_id }
    Assign->>DB: SELECT team_id FROM submissions WHERE id = submission_id
    Assign->>DB: SELECT team_id FROM team_members WHERE user_id = judge_id AND event_id = event_id
    Assign->>Assign: Check: judge not on submission's team? (I3)
    
    alt Conflict of interest
        Assign-->>O: 409 Conflict\n"Judge is a member of this submission's team"
    else No conflict
        Assign->>DB: INSERT judge_assignment (status=pending)
        Assign->>Audit: WRITE judge.assigned
        Assign-->>O: 201 Created { assignment_id }
    end
```

---

## Flow Diagram — Algorithmic Auto-Assignment

```mermaid
flowchart TD
    A["Organizer triggers\nPOST /events/{id}/assignments/auto"]
    B["Load all submitted, non-disqualified submissions"]
    C["Load all judges with role for this event"]
    D["Load team memberships for all judges"]
    E["For each submission S:"]
    F["Filter: exclude judges on S's team (I3)"]
    G["Sort eligible judges by\ncurrent assignment count (ASC)"]
    H["Assign top N judges\n(N = event.judges_per_submission)"]
    I["INSERT judge_assignment rows\n(status = pending)"]
    J{"All submissions\ncovered?"}
    K["Validate: every submission has\nexactly N assignments (I13)"]
    L["WRITE AuditLog: assignments.batch_created"]
    M["Return summary:\nN assignments created, X judges used"]

    A --> B
    B --> C
    C --> D
    D --> E
    E --> F
    F --> G
    G --> H
    H --> I
    I --> J
    J -->|Next submission| E
    J -->|Done| K
    K --> L
    L --> M
```

---

## Assignment Coverage Validation (I13)

Before the judging window can open, the system validates:

```
For every submission S where status = 'submitted' AND status != 'disqualified':
  count(JudgeAssignment WHERE submission_id = S.id AND status != 'recused') >= 1

If any submission has 0 assignments → block judging window from opening
```

The organizer sees a pre-flight checklist:
```
✅ 47 submissions covered (3 judges each)
✅ 141 total assignments created
✅ 12 judges, avg 11.75 assignments each
⚠️  Judge Alice: 8 assignments (below average — may need more)
```

---

## Error Cases

| Scenario | HTTP Status | Error Code |
|----------|-------------|------------|
| Judge on submission's team (I3) | 409 | `JUDGE_CONFLICT_OF_INTEREST` |
| Judge not invited to event | 403 | `NOT_A_JUDGE` |
| Duplicate assignment | 409 | `ASSIGNMENT_EXISTS` |
| Auto-assign during active judging | 422 | `JUDGING_IN_PROGRESS` |
| Not enough judges to cover all submissions | 422 | `INSUFFICIENT_JUDGES` |
