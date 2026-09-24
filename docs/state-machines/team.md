# State Machine: Team Lifecycle

> Teams are **event-scoped**. Once locked, membership cannot change.
> The `is_locked` flag protects submission integrity.

---

## State Diagram

```mermaid
stateDiagram-v2
    [*] --> forming : Team owner creates team\n(during registration_open or submissions_open)

    forming --> forming : Members join via invite_code\n(up to max_team_size)
    forming --> forming : Members leave team\n(owner cannot leave — must transfer or disband)

    forming --> locked : Team creates a submission\nOR submission_deadline_at passes

    locked --> locked : (No membership changes allowed)
    locked --> [*] : Event ends / archived
```

---

## Transition Rules

| From | To | Trigger | Guard |
|------|----|---------|-------|
| _(none)_ | `forming` | Team owner creates team | User registered for event (I1 checked — one team per user per event) |
| `forming` | `forming` (join) | Member uses invite_code | User registered for event; team not at `max_team_size`; user not on another team (I1) |
| `forming` | `forming` (leave) | Member leaves | Member is not the owner; team has no submitted submission |
| `forming` | `locked` | Team creates submission OR deadline passes | Automatic — invite_code deactivated |
| `locked` | — | No further transitions | Membership frozen |

---

## Invite Code Behaviour

```mermaid
sequenceDiagram
    participant Owner as Team Owner
    participant System as Platform
    participant Joiner as New Member

    Owner->>System: Creates team
    System-->>Owner: Returns unique invite_code
    Owner->>Joiner: Shares invite_code (out of band)
    Joiner->>System: POST /teams/join { invite_code }
    System->>System: Check: user registered for event?
    System->>System: Check: team not full?
    System->>System: Check: user not on another team? (I1)
    System-->>Joiner: Joined successfully
    System->>System: Write AuditLog: team.joined
```

---

## Lock Rules

The team locks (`is_locked = true`) when **either** of these happens first:
1. A team member calls "Create Submission" — the act of creating a draft locks the team.
2. `submission_deadline_at` passes — all teams with unsubmitted members are locked automatically.

Once locked:
- Invite code is invalidated
- No one can join
- No one can leave
- Team name can still be changed by owner (cosmetic only, no integrity impact)
