# Architecture & Flow Diagrams — Dogfood Hackathon Platform

> All diagrams use Mermaid syntax. Render in VS Code (Mermaid Preview extension), GitHub, or any Mermaid-compatible viewer.

---

## 1. System Architecture Overview

```mermaid
graph TB
    subgraph Client["Client Layer"]
        Browser["Browser\n(Next.js 14)"]
    end

    subgraph Gateway["API Gateway"]
        Chi["Chi v5 Router\n:8080\n+ JWT Middleware\n+ Rate Limiter\n+ Request Logger"]
    end

    subgraph Monolith["Go Modular Monolith — internal/"]
        Auth["auth/\nRegister · Login · JWT\nRefresh · Profile"]
        Events["events/\nCreate · List · Detail\nState Machine · Rubric · Judges"]
        Teams["teams/\nRegister · Create · Join · Leave"]
        Submissions["submissions/\nDraft · Submit · Upload"]
        Judging["judging/\nAssign · Score · Recuse · Normalize"]
        Voting["voting/\nCast · Retract · Results"]
        Admin["admin/\nUsers · Audit Log · Certificates"]
    end

    subgraph Infra["Infrastructure"]
        PG[("PostgreSQL 16\n19 tables")]
        Redis[("Redis 7\nJWT blacklist\nRate limits\nResults cache")]
        MinIO[("MinIO\nuploads/covers\nuploads/banners\ncertificates/")]
    end

    Browser -->|HTTPS REST| Chi
    Chi --> Auth
    Chi --> Events
    Chi --> Teams
    Chi --> Submissions
    Chi --> Judging
    Chi --> Voting
    Chi --> Admin

    Auth --> PG
    Auth --> Redis
    Events --> PG
    Teams --> PG
    Submissions --> PG
    Submissions --> MinIO
    Judging --> PG
    Voting --> PG
    Voting --> Redis
    Admin --> PG
```

---

## 2. Hexagonal Architecture — Single Module Layout

Every one of the 7 modules follows exactly this structure. Arrows = allowed import direction.

```mermaid
flowchart LR
    subgraph External["External World"]
        HTTP["HTTP Request\n(Chi router)"]
        DB["PostgreSQL\n(sqlx/pgx)"]
        Cache["Redis\n(go-redis)"]
        Store["MinIO\n(minio-go)"]
    end

    subgraph Module["internal/{module}/"]
        direction TB
        Handler["handler/\nHTTP-only\nReads r*http.Request\nWrites ResponseWriter\nCalls port/in.go interfaces"]

        subgraph Ports["port/"]
            PortIn["in.go\nUse Case interfaces\n+ Command structs"]
            PortOut["out.go\nRepository interfaces\nCache interfaces\nStorage interfaces"]
        end

        UseCase["usecase/\nOrchestration only\nCalls domain functions\nCalls port/out.go interfaces\nStarts transactions"]

        Domain["domain/\nPURE GO — zero framework imports\nStructs · Enums · Sentinel errors\nBusiness rule methods"]

        Repo["repository/\nSQL only — implements port/out.go\nNamedExecContext · GetContext\nRaw SQL for JOINs/CTEs"]
    end

    HTTP --> Handler
    Handler -->|calls| PortIn
    PortIn -->|implemented by| UseCase
    UseCase -->|calls| PortOut
    UseCase -->|calls| Domain
    PortOut -->|implemented by| Repo
    Repo --> DB

    style Domain fill:#1a472a,color:#fff
    style UseCase fill:#1e3a5f,color:#fff
    style Handler fill:#4a1942,color:#fff
    style Repo fill:#5c3317,color:#fff
```

---

## 3. Request Lifecycle — POST /events/{slug}/votes (Cast Vote)

```mermaid
sequenceDiagram
    participant Browser
    participant Chi as Chi Router
    participant JWTMiddleware as JWT Middleware
    participant VoteHandler as handler/VoteHandler
    participant CastVoteUC as usecase/CastVoteService
    participant VoteDomain as domain/Vote
    participant RateLimiter as cache/RedisRateLimiter
    participant VoteRepo as repository/PgVoteRepository
    participant AuditRepo as repository/PgAuditLogRepository
    participant PostgreSQL

    Browser->>Chi: POST /events/{slug}/votes\nAuthorization: Bearer {jwt}
    Chi->>JWTMiddleware: validate token
    JWTMiddleware-->>Chi: ctx with userID
    Chi->>VoteHandler: CastVote(w, r)

    Note over VoteHandler: Reads slug from URL param\nReads submissionID from body\nExtracts userID from ctx

    VoteHandler->>CastVoteUC: Cast(ctx, CastVoteCommand{...})

    CastVoteUC->>RateLimiter: Allow(ctx, userID, "vote")
    RateLimiter-->>CastVoteUC: ok / ErrRateLimited

    CastVoteUC->>VoteDomain: domain.HashIP(rawIP, salt)
    VoteDomain-->>CastVoteUC: ipHash

    Note over CastVoteUC: I12: Check voting window\nI5: DB UNIQUE enforces one vote

    CastVoteUC->>VoteRepo: Save(ctx, vote)
    VoteRepo->>PostgreSQL: INSERT INTO votes ...
    PostgreSQL-->>VoteRepo: ok / 23505 unique violation

    alt Unique violation (I5)
        VoteRepo-->>CastVoteUC: ErrDuplicate
        CastVoteUC-->>VoteHandler: ErrDuplicate
        VoteHandler-->>Browser: 409 DUPLICATE_RESOURCE
    else Success
        CastVoteUC->>AuditRepo: Write(ctx, AuditEntry{...})
        AuditRepo->>PostgreSQL: INSERT INTO audit_log ...
        CastVoteUC-->>VoteHandler: VoteDTO
        VoteHandler-->>Browser: 201 {"data": {...}, "meta": {...}}
    end
```

---

## 4. Event State Machine (Invariant I15)

```mermaid
stateDiagram-v2
    [*] --> draft : POST /events (organizer creates)

    draft --> registration_open : PATCH status=registration_open\n(organizer action)

    registration_open --> submissions_open : PATCH status=submissions_open\n(closes registration)

    submissions_open --> judging : PATCH status=judging\n(closes submission window)

    judging --> voting : PATCH status=voting\n(judges have scored)

    voting --> results_published : PATCH status=results_published\n(voting window closed)

    results_published --> archived : PATCH status=archived

    note right of draft
        No participants can register
        No teams can form
    end note

    note right of registration_open
        I3: Participants can register
        I6: Teams can be created
        I8: Can join/leave teams
    end note

    note right of submissions_open
        I9: Teams can submit
        I11: Submission editable
    end note

    note right of judging
        I12: Judges can score
        I16: Only own assignments
    end note

    note right of voting
        I12: Public can vote
        I5: One vote per user/submission
    end note
```

---

## 5. Database Entity Relationships (Core Tables)

```mermaid
erDiagram
    users {
        uuid id PK
        text email UK
        text display_name
        text password_hash
        bool is_admin
        timestamp created_at
    }

    events {
        uuid id PK
        text slug UK
        text title
        text status
        int max_team_size
        uuid organizer_id FK
        timestamp reg_opens_at
        timestamp reg_closes_at
        timestamp sub_opens_at
        timestamp sub_closes_at
        timestamp voting_opens_at
        timestamp voting_closes_at
    }

    tracks {
        uuid id PK
        uuid event_id FK
        text name
    }

    teams {
        uuid id PK
        uuid event_id FK
        uuid leader_id FK
        text name
        text invite_code UK
    }

    team_members {
        uuid team_id FK
        uuid user_id FK
        text role
    }

    submissions {
        uuid id PK
        uuid team_id FK
        uuid event_id FK
        uuid track_id FK
        text status
        text title
        text cover_url
        text repo_url
        timestamp submitted_at
    }

    rubrics {
        uuid id PK
        uuid event_id FK
        text name
    }

    rubric_criteria {
        uuid id PK
        uuid rubric_id FK
        text name
        decimal weight
        int max_score
    }

    judge_assignments {
        uuid id PK
        uuid judge_id FK
        uuid submission_id FK
        uuid event_id FK
        text status
    }

    scores {
        uuid id PK
        uuid assignment_id FK
        uuid criterion_id FK
        int raw_score
        decimal normalized_score
        text notes
    }

    votes {
        uuid id PK
        uuid submission_id FK
        uuid voter_id FK
        uuid event_id FK
        text ip_hash
        timestamp created_at
    }

    audit_log {
        uuid id PK
        uuid actor_id FK
        text action
        text entity_type
        uuid entity_id
        jsonb payload
        timestamp created_at
    }

    users ||--o{ events : "organizes"
    events ||--o{ tracks : "has"
    events ||--o{ teams : "has"
    teams ||--o{ team_members : "has"
    users ||--o{ team_members : "belongs to"
    teams ||--o| submissions : "submits"
    events ||--o{ submissions : "receives"
    tracks ||--o{ submissions : "categorizes"
    events ||--o| rubrics : "has"
    rubrics ||--o{ rubric_criteria : "has"
    submissions ||--o{ judge_assignments : "assigned to"
    users ||--o{ judge_assignments : "judges"
    judge_assignments ||--o{ scores : "has"
    rubric_criteria ||--o{ scores : "scored by"
    submissions ||--o{ votes : "receives"
    users ||--o{ votes : "casts"
    users ||--o{ audit_log : "generates"
```

---

## 6. Frontend Page Flow (User Journeys)

```mermaid
flowchart TD
    Start([User lands]) --> Home

    Home["/events\nEvent listing grid"] --> EventDetail

    EventDetail["/events/slug\nEvent detail + status"] --> |Not logged in| Login

    Login["/login"] --> |Success| EventDetail
    Login --> |No account| Register
    Register["/register"] --> |Success| EventDetail

    EventDetail --> |status=registration_open\nUser not registered| Participate

    Participate["/events/slug/participate\nStep 1: Register for event"] --> TeamSetup

    TeamSetup["Step 2: Team Setup\nCreate or Join"] --> |Create| TeamCreated
    TeamSetup --> |Join with code| TeamJoined
    TeamCreated --> TeamDash
    TeamJoined --> TeamDash

    TeamDash["Step 3: Team Dashboard\nInvite code + members"] --> |status=submissions_open| Submit

    Submit["/events/slug/submit\nSubmission editor\n+ file upload"] --> |Final submit| Gallery

    Gallery["/events/slug/submissions\nPublic gallery"] --> |status=voting| Vote

    Vote["/events/slug/vote\nVote on submissions"] --> Results

    Results["/events/slug/results\nLeaderboard + ranks"]

    OrganizerPath["Organizer\n/events/create"] --> OrgManage
    OrgManage["/events/slug/manage\nStatus · Rubric · Judges"] -.->|advances state| EventDetail

    JudgePath["Judge\n/judging\nAssignment queue"] --> JudgeScore
    JudgeScore["/judging/assignments/id\nScore each criterion"]

    AdminPath["Admin\n/admin/users\n/admin/audit-log\n/admin/certificates"]

    style Start fill:#1a472a,color:#fff
    style Results fill:#1e3a5f,color:#fff
    style Login fill:#4a1942,color:#fff
    style Register fill:#4a1942,color:#fff
```

---

## 7. Cross-Module Dependency Map

Only allowed imports are shown. Any other arrow = architecture violation.

```mermaid
graph LR
    subgraph Shared["internal/shared/"]
        Response["response/\nerror helpers\nApiResponse[T]"]
        Cache["cache/\nRedisRateLimiter"]
        Storage["storage/\nMinioFileStorage"]
        Middleware["middleware/\nJWT · Logger · CORS"]
    end

    subgraph Modules["internal/{module}/"]
        Auth["auth"]
        Events["events"]
        Teams["teams"]
        Subs["submissions"]
        Judging["judging"]
        Voting["voting"]
        Admin["admin"]
    end

    Wire["cmd/api/main.go\n(wiring only)"]

    Wire --> Auth
    Wire --> Events
    Wire --> Teams
    Wire --> Subs
    Wire --> Judging
    Wire --> Voting
    Wire --> Admin

    Teams -->|EventReader interface\nin teams/port/out.go| Events
    Subs -->|TeamReader interface\nin submissions/port/out.go| Teams
    Judging -->|SubmissionReader interface\nin judging/port/out.go| Subs
    Voting -->|SubmissionReader interface\nin voting/port/out.go| Subs
    Admin -->|UserReader interface\nin admin/port/out.go| Auth

    Auth --> Shared
    Events --> Shared
    Teams --> Shared
    Subs --> Shared
    Judging --> Shared
    Voting --> Shared
    Admin --> Shared

    style Wire fill:#333,color:#fff
    style Shared fill:#1e3a5f,color:#fff
```

---

## 8. Authentication Flow (JWT)

```mermaid
sequenceDiagram
    participant Browser
    participant Handler as handler/AuthHandler
    participant UseCase as usecase/LoginService
    participant UserRepo as repository/PgUserRepository
    participant TokenIssuer as shared/JWTIssuer
    participant BlacklistCache as cache/RedisCache

    Note over Browser: POST /auth/login

    Browser->>Handler: {email, password}
    Handler->>UseCase: Login(ctx, LoginCommand)

    UseCase->>UserRepo: FindByEmail(ctx, email)
    UserRepo-->>UseCase: User or ErrNotFound

    UseCase->>UseCase: bcrypt.CompareHashAndPassword()

    alt Password correct
        UseCase->>TokenIssuer: IssueTokenPair(userID, email, isAdmin)
        TokenIssuer-->>UseCase: {accessToken, refreshToken}
        UseCase->>UserRepo: SaveRefreshToken(ctx, token)
        UseCase-->>Handler: LoginDTO
        Handler-->>Browser: 200 {accessToken, refreshToken, user}
    else Wrong password
        UseCase-->>Handler: ErrUnauthorized
        Handler-->>Browser: 401 UNAUTHORIZED
    end

    Note over Browser: Subsequent requests

    Browser->>Handler: GET /events\nAuthorization: Bearer {accessToken}
    Handler->>BlacklistCache: IsRevoked(ctx, jti)
    BlacklistCache-->>Handler: false
    Handler->>Handler: Parse + validate JWT claims
    Handler->>Handler: Inject userID into ctx
```

---

## 9. Submission Lifecycle

```mermaid
stateDiagram-v2
    [*] --> draft : POST /events/{slug}/submissions\n(team creates draft)

    draft --> draft : PATCH /submissions/{id}\n(edit title, desc, track, URLs)

    draft --> draft : POST /submissions/{id}/files\n(upload cover image)

    draft --> submitted : POST /submissions/{id}/submit\n(I11: deadline not passed\nI13: must be draft status)

    submitted --> disqualified : POST /submissions/{id}/disqualify\n(organizer or admin only)

    note right of draft
        Editable
        Can upload cover
        Can change track
    end note

    note right of submitted
        Locked — no edits
        Visible in gallery
        Eligible for judging + voting
    end note

    note right of disqualified
        Hidden from public gallery
        Not scored or voted on
    end note
```
