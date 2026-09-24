# Flow 2 — Organizer Event Setup

> Covers the complete event configuration lifecycle from creation to publishing.
> **Invariants enforced**: I14, I15

---

## Flow Diagram

```mermaid
sequenceDiagram
    actor O as Organizer
    participant Event as Event Service
    participant Track as Track Service
    participant Rubric as Rubric Service
    participant DB as Database
    participant Audit as Audit Log

    %% Step 1: Create event (draft)
    O->>Event: POST /events { title, slug, description, dates... }
    Event->>Event: Validate: all required date windows present?
    Event->>Event: Validate: date ordering correct?\n(registration < submission < judging)
    Event->>DB: INSERT event (status=draft)
    Event->>Audit: WRITE event.created
    Event-->>O: 201 Created { event_id, slug }

    %% Step 2: Add tracks
    O->>Track: POST /events/{id}/tracks { name, description, prizes }
    Track->>DB: INSERT track
    Track-->>O: 201 Created { track_id }
    note over O,Track: Repeat for each track (e.g., "Open Track", "AI Track")

    %% Step 3: Create rubric
    O->>Rubric: POST /events/{id}/rubrics { name, description, track_id? }
    Rubric->>DB: INSERT rubric
    Rubric-->>O: 201 Created { rubric_id }

    %% Step 4: Add criteria to rubric
    O->>Rubric: POST /rubrics/{id}/criteria { name, max_score, weight, description }
    Rubric->>Rubric: Validate: sum of existing weights + new weight <= 1.0 (I14)
    Rubric->>DB: INSERT rubric_criterion
    Rubric-->>O: 201 Created
    note over O,Rubric: Repeat until weights sum to exactly 1.0

    O->>Rubric: POST /rubrics/{id}/criteria (last criterion)
    Rubric->>Rubric: Validate: total weight = 1.0? (I14)
    Rubric->>DB: INSERT rubric_criterion
    Rubric-->>O: 201 Created ✓

    %% Step 5: Configure voting (T3, optional)
    O->>Event: PATCH /events/{id} { voting_opens_at, voting_closes_at, voting_weight }
    Event->>DB: UPDATE event
    Event-->>O: 200 OK

    %% Step 6: Publish event
    O->>Event: POST /events/{id}/publish
    Event->>Event: Guard: event.status = draft? (I15)
    Event->>Event: Check: ≥1 track exists?
    Event->>Event: Check: ≥1 active rubric with weights = 1.0?
    Event->>Event: Check: all required dates set?
    Event->>DB: UPDATE event SET status=registration_open
    Event->>Audit: WRITE event.published
    Event-->>O: 200 OK { status: "registration_open" }
```

---

## Event Date Validation Rules

```mermaid
flowchart LR
    A["registration_opens_at"] -->|must be before| B["registration_closes_at"]
    B -->|must be before| C["submission_opens_at"]
    C -->|must be before| D["submission_deadline_at"]
    D -->|must be before| E["judging_opens_at"]
    E -->|must be before| F["judging_deadline_at"]
    F -->|must be before (if set)| G["voting_opens_at"]
    G -->|must be before| H["voting_closes_at"]
```

---

## Rubric Builder Logic

```mermaid
flowchart TD
    A["Organizer adds criterion\n{name, weight, max_score}"]
    B{"Sum of all weights\n= 1.0?"}
    C["Save criterion\nRubric marked: incomplete"]
    D["Save criterion\nRubric marked: complete ✓"]
    E{"Rubric complete?"}
    F["Block event publish"]
    G["Allow event publish"]

    A --> B
    B -->|No| C
    B -->|Yes| D
    C --> E
    D --> E
    E -->|No| F
    E -->|Yes| G
```

---

## Error Cases

| Scenario | HTTP Status | Error Code |
|----------|-------------|------------|
| Invalid date ordering | 422 | `INVALID_DATE_ORDER` |
| Weights exceed 1.0 (I14) | 422 | `RUBRIC_WEIGHT_EXCEEDED` |
| Publish with no tracks | 422 | `NO_TRACKS` |
| Publish with incomplete rubric | 422 | `RUBRIC_INCOMPLETE` |
| Invalid state transition (I15) | 409 | `INVALID_STATE_TRANSITION` |
| Slug already taken | 409 | `SLUG_TAKEN` |
