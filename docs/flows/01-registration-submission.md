# Flow 1 — Participant Registration & Submission

> Covers the complete journey from account creation through final project submission.
> **Invariants enforced**: I1, I2, I4, I19

---

## Flow Diagram

```mermaid
sequenceDiagram
    actor P as Participant
    participant Auth as Auth Service
    participant Event as Event Service
    participant Team as Team Service
    participant Sub as Submission Service
    participant DB as Database
    participant Audit as Audit Log

    %% Step 1: Register account
    P->>Auth: POST /auth/register { email, password, display_name }
    Auth->>DB: INSERT user (is_verified=false)
    Auth-->>P: 201 Created + verification email sent

    P->>Auth: GET /auth/verify?token=...
    Auth->>DB: UPDATE user SET is_verified=true
    Auth-->>P: 200 OK + JWT issued

    %% Step 2: Register for event
    P->>Event: POST /events/{slug}/register
    Event->>Event: Check: event.status = registration_open OR submissions_open?
    Event->>DB: INSERT event_registration
    Event->>Audit: WRITE user.event_registered
    Event-->>P: 201 Created (participant role granted)

    %% Step 3a: Create a team
    P->>Team: POST /events/{slug}/teams { name }
    Team->>Team: Check: user registered for event?
    Team->>Team: Check: user not already on a team? (I1)
    Team->>DB: INSERT team + team_member (role=owner)
    Team->>Audit: WRITE team.created
    Team-->>P: 201 Created { invite_code }

    %% Step 3b: OR join a team
    note over P,Team: Alternative: join via invite code
    P->>Team: POST /teams/join { invite_code }
    Team->>Team: Check: team not locked?
    Team->>Team: Check: team not at max_team_size?
    Team->>Team: Check: user not on another team? (I1)
    Team->>DB: INSERT team_member (role=member)
    Team->>Audit: WRITE team.joined
    Team-->>P: 200 OK

    %% Step 4: Create draft submission
    P->>Sub: POST /events/{slug}/submissions { title, track_id, ... }
    Sub->>Sub: Check: event.status = submissions_open?
    Sub->>Sub: Check: no existing submission for this team? (I2)
    Sub->>DB: INSERT submission (status=draft)
    Sub->>DB: UPDATE team SET is_locked=true
    Sub->>Audit: WRITE submission.created
    Sub-->>P: 201 Created { submission_id }

    %% Step 5: Edit submission (repeatable)
    P->>Sub: PATCH /submissions/{id} { description, demo_url, repo_url, ... }
    Sub->>Sub: Check: submission.status = draft?
    Sub->>Sub: Check: now < event.submission_deadline_at? (I4)
    Sub->>DB: UPDATE submission fields
    Sub->>Audit: WRITE submission.edited
    Sub-->>P: 200 OK

    %% Step 6: Submit final
    P->>Sub: POST /submissions/{id}/submit
    Sub->>Sub: Check: status = draft?
    Sub->>Sub: Check: now < event.submission_deadline_at? (I4)
    Sub->>Sub: Check: required fields present? (title, track, demo/repo url)
    Sub->>DB: UPDATE submission SET status=submitted, submitted_at=now()
    Sub->>Audit: WRITE submission.submitted
    Sub-->>P: 200 OK { submitted_at }
```

---

## Un-registration Flow

```mermaid
sequenceDiagram
    actor P as Participant
    participant Event as Event Service
    participant DB as Database
    participant Audit as Audit Log

    P->>Event: DELETE /events/{slug}/register
    Event->>Event: Check: event.status allows un-registration?
    Event->>DB: SELECT submission WHERE team has this user
    Event->>Event: Check: submission.status != 'submitted'? (I19)
    alt Has submitted project
        Event-->>P: 409 Conflict\n"Cannot un-register with a submitted project"
    else No submitted project (draft or no submission)
        Event->>DB: UPDATE event_registration SET is_active=false, unregistered_at=now()
        Event->>Audit: WRITE user.event_unregistered
        Event-->>P: 200 OK
    end
```

---

## Error Cases & Responses

| Scenario | HTTP Status | Error Code |
|----------|-------------|------------|
| Email already registered | 409 | `EMAIL_TAKEN` |
| Event not in registration window | 422 | `EVENT_NOT_OPEN` |
| User already on a team (I1) | 409 | `ALREADY_IN_TEAM` |
| Team at max capacity | 422 | `TEAM_FULL` |
| Duplicate submission (I2) | 409 | `SUBMISSION_EXISTS` |
| Edit after deadline (I4) | 403 | `SUBMISSION_LOCKED` |
| Submit after deadline (I4) | 403 | `SUBMISSION_DEADLINE_PASSED` |
| Un-register with submitted project (I19) | 409 | `CANNOT_UNREGISTER_SUBMITTED` |
