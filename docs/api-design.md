# API Design — Dogfood Hackathon Platform
# Version: 1.0 | Written: September 2026
# This is the contract. Frontend, Backend, and Acceptance Suite all implement against this.

---

## Global Conventions

### Base URL
```
http://localhost/api/v1         (local dev via nginx)
http://api.dogfood.local/api/v1 (docker compose internal)
```

### Authentication
All authenticated endpoints require:
```
Authorization: Bearer <access_token>
```
Access token: HS256 JWT, 24hr expiry.
Refresh token: httpOnly cookie, 7d expiry.

### Standard Response Envelope
Every response — success or error — is wrapped.

**Success (single object):**
```json
{
  "data": { ... },
  "meta": {
    "requestId": "uuid-v4",
    "timestamp": "2026-09-14T08:00:00Z"
  }
}
```

**Success (list / paginated):**
```json
{
  "data": [ ... ],
  "meta": {
    "requestId": "uuid-v4",
    "timestamp": "2026-09-14T08:00:00Z",
    "page": 1,
    "pageSize": 20,
    "totalPages": 5,
    "totalCount": 92
  }
}
```

**Error:**
```json
{
  "error": {
    "code": "MACHINE_READABLE_CODE",
    "message": "Human-readable description of what went wrong"
  },
  "meta": {
    "requestId": "uuid-v4",
    "timestamp": "2026-09-14T08:00:00Z"
  }
}
```

### Standard Error Codes

| HTTP Status | Error Code | When |
|-------------|-----------|------|
| 400 | `VALIDATION_ERROR` | Request body fails Bean Validation |
| 401 | `UNAUTHORIZED` | Missing or invalid JWT |
| 403 | `FORBIDDEN` | Authenticated but wrong role for this event |
| 404 | `NOT_FOUND` | Resource does not exist |
| 409 | `DUPLICATE_RESOURCE` | Unique constraint violation (email, one team per event, etc.) |
| 422 | `DEADLINE_PASSED` | Action attempted after the relevant deadline |
| 422 | `INVALID_STATE_TRANSITION` | Event/submission status machine rejected transition |
| 422 | `INVARIANT_VIOLATION` | Business rule violation (judge scoring own team, etc.) |
| 429 | `RATE_LIMITED` | Rate limit exceeded |
| 500 | `INTERNAL_ERROR` | Unhandled server error |

### Pagination
All list endpoints accept:
```
?page=1&pageSize=20
```
Default: `page=1`, `pageSize=20`, max `pageSize=100`.

### Date Format
All dates are ISO 8601 UTC: `"2026-09-14T08:00:00Z"`

### Slug Convention
Events are referenced by `slug` in URLs (e.g., `hackathon-2026`), not by UUID.
All other resources use UUID.

---

## Module 1: Auth

### POST /auth/register
Create a new user account.

**Auth required:** No

**Request:**
```json
{
  "email": "alice@example.com",
  "password": "Min8CharsAtLeast1Number",
  "displayName": "Alice Chen",
  "captchaToken": "xyz..." 
}
```

**Validation:**
- `email`: valid email format, max 255 chars
- `password`: min 8 chars, max 72 chars, must not be compromised (checked against HIBP)
- `displayName`: min 2 chars, max 80 chars
- `captchaToken`: valid CAPTCHA token (required for A-008)

**Response 201:**
```json
{
  "data": {
    "userId": "uuid",
    "email": "alice@example.com",
    "displayName": "Alice Chen",
    "createdAt": "2026-09-14T08:00:00Z"
  }
}
```

**Errors:**
| Code | Condition |
|------|-----------|
| `DUPLICATE_RESOURCE` 409 | Email already registered (I1 enforcement) |
| `VALIDATION_ERROR` 400 | Invalid email / weak or compromised password / invalid captcha |
| `RATE_LIMITED` 429 | >5 requests per IP in 15 minutes |

---

### POST /auth/login
Authenticate and receive tokens.

**Auth required:** No

**Request:**
```json
{
  "email": "alice@example.com",
  "password": "Min8CharsAtLeast1Number"
}
```

**Response 200:**
```json
{
  "data": {
    "accessToken": "eyJhbGci...",
    "tokenType": "Bearer",
    "expiresIn": 86400,
    "user": {
      "userId": "uuid",
      "email": "alice@example.com",
      "displayName": "Alice Chen",
      "isAdmin": false
    }
  }
}
```
Also sets `Set-Cookie: refresh_token=<token>; HttpOnly; SameSite=Strict; Max-Age=604800`

**Errors:**
| Code | Condition |
|------|-----------|
| `UNAUTHORIZED` 401 | Wrong email or password (same message, no enumeration), or email not verified |
| `RATE_LIMITED` 429 | >5 requests per IP in 15 minutes |

---

### POST /auth/refresh
Exchange refresh token for a new access token.

**Auth required:** No (uses the refresh token in the request body)

**Request:**
```json
{ "refreshToken": "opaque-refresh-token" }
```

**Response 200:**
```json
{
  "data": {
    "accessToken": "eyJhbGci...",
    "refreshToken": "rotated-refresh-token",
    "user": {
      "id": "uuid",
      "email": "alice@example.com",
      "displayName": "Alice Chen",
      "avatarUrl": null,
      "isAdmin": false,
      "createdAt": "2026-09-14T08:00:00Z"
    }
  }
}
```

**Errors:**
| Code | Condition |
|------|-----------|
| `UNAUTHORIZED` 401 | Refresh token expired, revoked, or not found |

---

### POST /auth/logout
Invalidate the current session.

**Auth required:** Yes (any authenticated user)

**Request:** No body.

**Response 204:** No content.

**Behavior:**
- Adds JWT `jti` to Redis blacklist with TTL = remaining token lifetime
- Revokes the refresh token in PostgreSQL

---

### POST /auth/verify-email
Verify an email address using a token (Upcoming for A-006).

**Request:**
```json
{
  "token": "abc123def456"
}
```

---

### Passkey Authentication (Upcoming for A-007)
- `POST /auth/passkeys/register/start`
- `POST /auth/passkeys/register/finish`
- `POST /auth/passkeys/login/start`
- `POST /auth/passkeys/login/finish`

---

## Module 2: Events

### POST /events
Create a new event. Caller becomes the organizer.

**Auth required:** Yes (any authenticated user becomes organizer)

**Request:**
```json
{
  "name": "Hackathon 2026",
  "slug": "hackathon-2026",
  "description": "Annual global hackathon",
  "registrationStart": "2026-10-01T00:00:00Z",
  "registrationEnd": "2026-10-15T23:59:59Z",
  "submissionDeadline": "2026-10-20T23:59:59Z",
  "judgingStart": "2026-10-21T00:00:00Z",
  "judgingEnd": "2026-10-25T23:59:59Z",
  "votingStart": "2026-10-26T00:00:00Z",
  "votingEnd": "2026-10-28T23:59:59Z",
  "maxTeamSize": 4,
  "tracks": ["AI/ML", "Web3", "Open Innovation"],
  "coverImageUrl": null
}
```

**Validation:**
- `slug`: lowercase alphanumeric + hyphens, unique, max 80 chars
- `registrationEnd` must be after `registrationStart`
- `submissionDeadline` must be after `registrationEnd`
- `judgingStart` must be after `submissionDeadline`
- `maxTeamSize`: 1–10
- `tracks`: at least 1, max 10, each max 60 chars

**Response 201:**
```json
{
  "data": {
    "eventId": "uuid",
    "slug": "hackathon-2026",
    "name": "Hackathon 2026",
    "status": "draft",
    "organizerId": "uuid",
    "registrationStart": "2026-10-01T00:00:00Z",
    "registrationEnd": "2026-10-15T23:59:59Z",
    "submissionDeadline": "2026-10-20T23:59:59Z",
    "judgingStart": "2026-10-21T00:00:00Z",
    "judgingEnd": "2026-10-25T23:59:59Z",
    "votingStart": "2026-10-26T00:00:00Z",
    "votingEnd": "2026-10-28T23:59:59Z",
    "maxTeamSize": 4,
    "tracks": ["AI/ML", "Web3", "Open Innovation"],
    "normalizationStatus": "not_started",
    "createdAt": "2026-09-14T08:00:00Z"
  }
}
```

**Errors:**
| Code | Condition |
|------|-----------|
| `DUPLICATE_RESOURCE` 409 | Slug already taken |
| `VALIDATION_ERROR` 400 | Invalid date ordering, invalid slug format |

---

### GET /events
List all public events (paginated).

**Auth required:** No

**Query params:** `?page=1&pageSize=20&status=registration_open`

**Response 200:**
```json
{
  "data": [
    {
      "eventId": "uuid",
      "slug": "hackathon-2026",
      "name": "Hackathon 2026",
      "status": "registration_open",
      "submissionDeadline": "2026-10-20T23:59:59Z",
      "trackCount": 3,
      "registeredTeamCount": 47
    }
  ],
  "meta": { "page": 1, "pageSize": 20, "totalPages": 1, "totalCount": 3 }
}
```

Note: `draft` status events are excluded for non-admins.

---

### GET /events/{slug}
Get full event details.

**Auth required:** No (public). Authenticated users also receive their own role in this event.

**Response 200:**
```json
{
  "data": {
    "eventId": "uuid",
    "slug": "hackathon-2026",
    "name": "Hackathon 2026",
    "description": "Annual global hackathon",
    "status": "registration_open",
    "registrationStart": "2026-10-01T00:00:00Z",
    "registrationEnd": "2026-10-15T23:59:59Z",
    "submissionDeadline": "2026-10-20T23:59:59Z",
    "judgingStart": "2026-10-21T00:00:00Z",
    "judgingEnd": "2026-10-25T23:59:59Z",
    "votingStart": "2026-10-26T00:00:00Z",
    "votingEnd": "2026-10-28T23:59:59Z",
    "maxTeamSize": 4,
    "tracks": [
      { "trackId": "uuid", "name": "AI/ML" },
      { "trackId": "uuid", "name": "Web3" }
    ],
    "normalizationStatus": "not_started",
    "myRole": "participant",
    "myTeamId": "uuid-or-null",
    "mySubmissionId": "uuid-or-null",
    "rubric": {
      "criteria": [
        { "criterionId": "uuid", "name": "Innovation", "weight": 0.40 },
        { "criterionId": "uuid", "name": "Technical Depth", "weight": 0.35 },
        { "criterionId": "uuid", "name": "Presentation", "weight": 0.25 }
      ]
    }
  }
}
```

**Errors:**
| Code | Condition |
|------|-----------|
| `NOT_FOUND` 404 | Slug does not exist |

---

### PATCH /events/{slug}
Update event details. Only in `draft` status.

**Auth required:** Yes — `organizer` role for this event

**Request (all fields optional):**
```json
{
  "name": "Updated Name",
  "description": "Updated description",
  "registrationStart": "2026-10-02T00:00:00Z",
  "maxTeamSize": 5
}
```

**Response 200:** Full event object (same as GET /events/{slug} `data`).

**Errors:**
| Code | Condition |
|------|-----------|
| `FORBIDDEN` 403 | Not the organizer |
| `INVALID_STATE_TRANSITION` 422 | Event is not in `draft` status |
| `VALIDATION_ERROR` 400 | Invalid date ordering |

---

### POST /events/{slug}/status
Transition event to a new status. Enforces I15 state machine.

**Auth required:** Yes — `organizer` role for this event

**Request:**
```json
{
  "status": "registration_open"
}
```

Valid transitions (I15):
```
draft → registration_open
registration_open → submission_open
submission_open → judging
judging → results_ready (only if normalization_status = complete)
results_ready → results_published
Any → archived
```

**Response 200:**
```json
{
  "data": {
    "slug": "hackathon-2026",
    "previousStatus": "draft",
    "currentStatus": "registration_open",
    "transitionedAt": "2026-09-14T08:00:00Z"
  }
}
```

**Errors:**
| Code | Condition |
|------|-----------|
| `FORBIDDEN` 403 | Not the organizer |
| `INVALID_STATE_TRANSITION` 422 | Transition not in the allowed graph |
| `INVARIANT_VIOLATION` 422 | Trying results_ready but normalization not complete |

---

### POST /events/{slug}/rubric
Define or replace the judging rubric for this event.

**Auth required:** Yes — `organizer` role

**Request:**
```json
{
  "criteria": [
    { "name": "Innovation", "weight": 0.40 },
    { "name": "Technical Depth", "weight": 0.35 },
    { "name": "Presentation", "weight": 0.25 }
  ]
}
```

**Validation (I14):**
- At least 1 criterion, max 10
- All weights must sum to exactly 1.0 (tolerance: ±0.001)
- Each weight > 0

**Response 201:**
```json
{
  "data": {
    "rubricId": "uuid",
    "eventSlug": "hackathon-2026",
    "criteria": [
      { "criterionId": "uuid", "name": "Innovation", "weight": 0.40 },
      { "criterionId": "uuid", "name": "Technical Depth", "weight": 0.35 },
      { "criterionId": "uuid", "name": "Presentation", "weight": 0.25 }
    ],
    "createdAt": "2026-09-14T08:00:00Z"
  }
}
```

**Errors:**
| Code | Condition |
|------|-----------|
| `FORBIDDEN` 403 | Not the organizer |
| `INVARIANT_VIOLATION` 422 | Weights do not sum to 1.0 (I14) |

---

### POST /events/{slug}/judges
Assign a registered user as judge for this event.

**Auth required:** Yes — `organizer` role

**Request:**
```json
{
  "userId": "uuid"
}
```

**Response 201:**
```json
{
  "data": {
    "judgeId": "uuid",
    "displayName": "Bob Smith",
    "email": "bob@example.com",
    "assignedAt": "2026-09-14T08:00:00Z"
  }
}
```

**Errors:**
| Code | Condition |
|------|-----------|
| `FORBIDDEN` 403 | Not the organizer |
| `NOT_FOUND` 404 | User does not exist |
| `DUPLICATE_RESOURCE` 409 | User already a judge for this event |

---

### DELETE /events/{slug}/judges/{judgeId}
Remove a judge from the event.

**Auth required:** Yes — `organizer` role

**Response 204:** No content.

---

## Module 3: Event Registration & Teams

### POST /events/{slug}/register
Register authenticated user for an event.

**Auth required:** Yes

**Request:** No body.

**Response 201:**
```json
{
  "data": {
    "registrationId": "uuid",
    "eventSlug": "hackathon-2026",
    "userId": "uuid",
    "registeredAt": "2026-09-14T08:00:00Z"
  }
}
```

**Errors:**
| Code | Condition |
|------|-----------|
| `DEADLINE_PASSED` 422 | Registration window closed |
| `INVALID_STATE_TRANSITION` 422 | Event not in `registration_open` status |
| `DUPLICATE_RESOURCE` 409 | Already registered |

---

### DELETE /events/{slug}/register
Un-register from an event.

**Auth required:** Yes — must be registered participant, no locked team

**Response 200:**
```json
{ "data": { "message": "Successfully un-registered" } }
```

**Errors:**
| Code | Condition |
|------|-----------|
| `INVARIANT_VIOLATION` 422 | Team is locked (submission exists) — I19 |
| `DEADLINE_PASSED` 422 | Submission deadline passed |

---

### POST /events/{slug}/teams
Create a new team for this event. Caller becomes owner.

**Auth required:** Yes — registered participant for this event

**Request:**
```json
{
  "name": "Team Rocket",
  "trackId": "uuid"
}
```

**Response 201:**
```json
{
  "data": {
    "teamId": "uuid",
    "name": "Team Rocket",
    "inviteCode": "XKCD-4729",
    "track": { "trackId": "uuid", "name": "AI/ML" },
    "members": [
      { "userId": "uuid", "displayName": "Alice Chen", "role": "owner" }
    ],
    "isLocked": false,
    "createdAt": "2026-09-14T08:00:00Z"
  }
}
```

**Errors:**
| Code | Condition |
|------|-----------|
| `INVARIANT_VIOLATION` 422 | User already on a team in this event (I1) |
| `DEADLINE_PASSED` 422 | Registration window closed |

---

### GET /events/{slug}/teams/mine
Get the authenticated user's team for this event.

**Auth required:** Yes — registered participant

**Response 200:** Same shape as POST /events/{slug}/teams response `data`.

**Errors:**
| Code | Condition |
|------|-----------|
| `NOT_FOUND` 404 | User not on a team for this event |

---

### POST /teams/join
Join a team using an invite code.

**Auth required:** Yes — registered participant for the relevant event

**Request:**
```json
{
  "inviteCode": "XKCD-4729"
}
```

**Response 200:**
```json
{
  "data": {
    "teamId": "uuid",
    "name": "Team Rocket",
    "eventSlug": "hackathon-2026",
    "members": [ ... ]
  }
}
```

**Errors:**
| Code | Condition |
|------|-----------|
| `NOT_FOUND` 404 | Invite code does not match any open team |
| `INVARIANT_VIOLATION` 422 | User already on a team in this event (I1) |
| `INVARIANT_VIOLATION` 422 | Team at max_team_size |
| `INVARIANT_VIOLATION` 422 | Team is locked (submission already created) |

---

### DELETE /teams/{teamId}/members/me
Leave a team (non-owners only; owners must disband or transfer).

**Auth required:** Yes — team member

**Response 200:**
```json
{ "data": { "message": "Successfully left the team" } }
```

**Errors:**
| Code | Condition |
|------|-----------|
| `INVARIANT_VIOLATION` 422 | Team is locked (I2) — cannot leave after creating submission |
| `FORBIDDEN` 403 | User is the owner (owners cannot leave, only disband) |

---

## Module 4: Submissions

### POST /events/{slug}/submissions
Create a draft submission. Locks the team.

**Auth required:** Yes — team owner for a team in this event

**Request:**
```json
{
  "title": "TerraBot — AI for Sustainable Farming",
  "description": "Our project uses computer vision to...",
  "repoUrl": "https://github.com/team-rocket/terrabot",
  "demoUrl": "https://terrabot.demo.example.com",
  "videoUrl": "https://youtube.com/watch?v=..."
}
```

**Validation:**
- `title`: max 200 chars
- `description`: max 5000 chars
- `repoUrl`, `demoUrl`, `videoUrl`: valid URL format, optional

**Response 201:**
```json
{
  "data": {
    "submissionId": "uuid",
    "title": "TerraBot — AI for Sustainable Farming",
    "status": "draft",
    "teamId": "uuid",
    "teamName": "Team Rocket",
    "eventSlug": "hackathon-2026",
    "track": { "trackId": "uuid", "name": "AI/ML" },
    "repoUrl": "https://github.com/team-rocket/terrabot",
    "demoUrl": null,
    "videoUrl": null,
    "coverImageUrl": null,
    "createdAt": "2026-09-14T08:00:00Z",
    "updatedAt": "2026-09-14T08:00:00Z"
  }
}
```

**Errors:**
| Code | Condition |
|------|-----------|
| `DUPLICATE_RESOURCE` 409 | Team already has a submission for this event (I2) |
| `FORBIDDEN` 403 | Not the team owner |
| `DEADLINE_PASSED` 422 | Submission window closed |

---

### PATCH /submissions/{submissionId}
Update a draft submission.

**Auth required:** Yes — team owner, submission must be in `draft` status

**Request (all fields optional):**
```json
{
  "title": "TerraBot v2",
  "description": "Updated description...",
  "repoUrl": "https://github.com/team-rocket/terrabot-v2",
  "demoUrl": "https://demo.terrabot.io",
  "videoUrl": null
}
```

**Response 200:** Full submission object (same as POST response `data`).

**Errors:**
| Code | Condition |
|------|-----------|
| `FORBIDDEN` 403 | Not the team owner |
| `INVALID_STATE_TRANSITION` 422 | Submission is not in `draft` status |
| `DEADLINE_PASSED` 422 | Submission deadline passed |

---

### POST /submissions/{submissionId}/submit
Finalize a submission (draft → submitted). Cannot be undone.

**Auth required:** Yes — team owner

**Request:** No body.

**Response 200:**
```json
{
  "data": {
    "submissionId": "uuid",
    "status": "submitted",
    "submittedAt": "2026-10-20T22:47:33Z"
  }
}
```

**Errors:**
| Code | Condition |
|------|-----------|
| `FORBIDDEN` 403 | Not the team owner |
| `DEADLINE_PASSED` 422 | Submission deadline passed |
| `INVALID_STATE_TRANSITION` 422 | Already submitted |

---

### POST /submissions/{submissionId}/files
Upload a file attachment (cover image, banner, etc.).

**Auth required:** Yes — team owner

**Request:** `multipart/form-data`
```
file:     <binary>
fileType: "cover_image" | "banner" | "attachment"
```

**Limits:**
- `cover_image`: max 5MB, JPEG/PNG only
- `attachment`: max 20MB, any type

**Response 201:**
```json
{
  "data": {
    "uploadId": "uuid",
    "fileType": "cover_image",
    "url": "http://localhost:9000/uploads/submissions/uuid/cover.jpg",
    "sizeBytes": 204800,
    "uploadedAt": "2026-09-14T08:00:00Z"
  }
}
```

**Errors:**
| Code | Condition |
|------|-----------|
| `VALIDATION_ERROR` 400 | File too large or wrong MIME type |
| `FORBIDDEN` 403 | Not the team owner |
| `DEADLINE_PASSED` 422 | Submission deadline passed |

---

### GET /events/{slug}/submissions
Get paginated gallery of all submitted projects for an event.

**Auth required:** No (public after `results_published`). Authenticated organizers/judges see it during judging.

**Query params:** `?page=1&pageSize=20&trackId=uuid&sort=votes_desc`

**Response 200:**
```json
{
  "data": [
    {
      "submissionId": "uuid",
      "title": "TerraBot",
      "description": "Our project uses...",
      "teamName": "Team Rocket",
      "track": { "trackId": "uuid", "name": "AI/ML" },
      "coverImageUrl": "http://...",
      "repoUrl": "https://github.com/...",
      "demoUrl": "https://...",
      "normalizedScore": 87.4,
      "rank": 1,
      "voteCount": 142
    }
  ],
  "meta": { "page": 1, "pageSize": 20, "totalPages": 3, "totalCount": 47 }
}
```

Note: `normalizedScore` and `rank` are `null` until `results_published`.
Note: `voteCount` is `null` during the voting window (I12).

---

### POST /submissions/{submissionId}/disqualify
Disqualify a submission.

**Auth required:** Yes — `organizer` role for this event

**Request:**
```json
{
  "reason": "Violated rule 4.3: pre-existing codebase submitted"
}
```

**Response 200:**
```json
{
  "data": {
    "submissionId": "uuid",
    "status": "disqualified",
    "disqualifiedAt": "2026-10-22T10:15:00Z",
    "reason": "Violated rule 4.3: pre-existing codebase submitted"
  }
}
```

---

## Module 5: Judging

### GET /judge/queue
Get all pending judging assignments for the authenticated judge.

**Auth required:** Yes — `judge` role for at least one event

**Query params:** `?eventSlug=hackathon-2026`

**Response 200:**
```json
{
  "data": [
    {
      "assignmentId": "uuid",
      "submissionId": "uuid",
      "submissionTitle": "TerraBot",
      "teamName": "Team Rocket",
      "track": { "trackId": "uuid", "name": "AI/ML" },
      "eventSlug": "hackathon-2026",
      "status": "pending",
      "assignedAt": "2026-10-21T00:00:00Z"
    }
  ]
}
```

---

### GET /judge/assignments/{assignmentId}
Get full detail of one judging assignment including the rubric.

**Auth required:** Yes — the assigned judge only (I9)

**Response 200:**
```json
{
  "data": {
    "assignmentId": "uuid",
    "status": "pending",
    "submission": {
      "submissionId": "uuid",
      "title": "TerraBot",
      "description": "Our project uses...",
      "repoUrl": "https://github.com/...",
      "demoUrl": "https://...",
      "videoUrl": "https://...",
      "teamName": "Team Rocket",
      "track": { "trackId": "uuid", "name": "AI/ML" }
    },
    "rubric": {
      "criteria": [
        { "criterionId": "uuid", "name": "Innovation", "weight": 0.40 },
        { "criterionId": "uuid", "name": "Technical Depth", "weight": 0.35 },
        { "criterionId": "uuid", "name": "Presentation", "weight": 0.25 }
      ]
    },
    "scores": null,
    "assignedAt": "2026-10-21T00:00:00Z"
  }
}
```

**Errors:**
| Code | Condition |
|------|-----------|
| `FORBIDDEN` 403 | Not the assigned judge (I9) |
| `NOT_FOUND` 404 | Assignment not found |

---

### POST /judge/assignments/{assignmentId}/scores
Submit scores for an assignment.

**Auth required:** Yes — the assigned judge only

**Request:**
```json
{
  "scores": [
    { "criterionId": "uuid", "score": 8.5, "comment": "Strong originality" },
    { "criterionId": "uuid", "score": 7.0, "comment": "Good but lacks depth" },
    { "criterionId": "uuid", "score": 9.0, "comment": "Excellent demo" }
  ]
}
```

**Validation:**
- Each `score`: 0.0 – 10.0 (inclusive), 1 decimal place
- Must provide exactly one score per rubric criterion
- `comment`: optional, max 500 chars

**Response 201:**
```json
{
  "data": {
    "assignmentId": "uuid",
    "status": "completed",
    "weightedScore": 8.275,
    "submittedAt": "2026-10-22T14:33:00Z"
  }
}
```

**Errors:**
| Code | Condition |
|------|-----------|
| `FORBIDDEN` 403 | Not the assigned judge (I9) |
| `INVARIANT_VIOLATION` 422 | Judge is a member of the submission's team (I3) |
| `INVARIANT_VIOLATION` 422 | Assignment already completed (I8) |
| `VALIDATION_ERROR` 400 | Missing criteria, score out of range |

---

### POST /judge/assignments/{assignmentId}/recuse
Recuse from an assignment (e.g., conflict of interest).

**Auth required:** Yes — the assigned judge

**Request:**
```json
{
  "reason": "I know this team personally"
}
```

**Response 200:**
```json
{
  "data": {
    "assignmentId": "uuid",
    "status": "recused",
    "recusedAt": "2026-10-21T09:00:00Z"
  }
}
```

---

### POST /events/{slug}/normalize
Trigger the Z-score normalization job. Runs asynchronously.

**Auth required:** Yes — `organizer` role

**Request:** No body.

**Response 202:** Accepted (job started in background)
```json
{
  "data": {
    "eventSlug": "hackathon-2026",
    "normalizationStatus": "in_progress",
    "triggeredAt": "2026-10-25T16:00:00Z",
    "message": "Normalization job started. Poll GET /events/{slug} for normalizationStatus."
  }
}
```

**Errors:**
| Code | Condition |
|------|-----------|
| `FORBIDDEN` 403 | Not the organizer |
| `INVALID_STATE_TRANSITION` 422 | Event not in `judging` status |
| `INVARIANT_VIOLATION` 422 | Not all judge assignments are complete |

Poll `GET /events/{slug}` → `data.normalizationStatus` until `complete` or `failed`.

---

## Module 6: Voting

### POST /events/{slug}/votes
Cast a vote for a submission. One vote per user per event (upsert).

**Auth required:** Yes — registered participant for this event

**Request:**
```json
{
  "submissionId": "uuid"
}
```

**Behavior:** Upsert — if the user already voted, replaces their previous vote.

**Response 201:**
```json
{
  "data": {
    "voteId": "uuid",
    "submissionId": "uuid",
    "submissionTitle": "TerraBot",
    "eventSlug": "hackathon-2026",
    "votedAt": "2026-10-27T11:22:00Z"
  }
}
```

**Errors:**
| Code | Condition |
|------|-----------|
| `INVARIANT_VIOLATION` 422 | Voting window not open (I12) |
| `INVARIANT_VIOLATION` 422 | User voting for their own team's submission (I5) |
| `NOT_FOUND` 404 | Submission does not exist or not in this event |
| `RATE_LIMITED` 429 | >5 vote changes in 10 minutes |

---

### GET /events/{slug}/results
Get the public leaderboard / results.

**Auth required:** No

**Response 200:**
```json
{
  "data": {
    "eventSlug": "hackathon-2026",
    "status": "results_published",
    "publishedAt": "2026-10-29T10:00:00Z",
    "leaderboard": [
      {
        "rank": 1,
        "submissionId": "uuid",
        "title": "TerraBot",
        "teamName": "Team Rocket",
        "track": { "trackId": "uuid", "name": "AI/ML" },
        "normalizedScore": 91.2,
        "voteCount": 142,
        "certificateUrl": "/api/v1/certificates/verify/abc123hash"
      },
      {
        "rank": 2,
        "submissionId": "uuid",
        "title": "ChainGuard",
        "teamName": "Blockchain Badgers",
        "track": { "trackId": "uuid", "name": "Web3" },
        "normalizedScore": 87.4,
        "voteCount": 98,
        "certificateUrl": null
      }
    ]
  }
}
```

**Errors:**
| Code | Condition |
|------|-----------|
| `INVARIANT_VIOLATION` 422 | Results not yet published (event not in `results_published` status) |

---

## Module 7: Admin

### GET /admin/users
List all users on the platform.

**Auth required:** Yes — `is_admin = true`

**Query params:** `?page=1&pageSize=20&email=alice@example.com`

**Response 200:**
```json
{
  "data": [
    {
      "userId": "uuid",
      "email": "alice@example.com",
      "displayName": "Alice Chen",
      "isAdmin": false,
      "createdAt": "2026-09-14T08:00:00Z",
      "lastLoginAt": "2026-09-20T11:00:00Z"
    }
  ],
  "meta": { "page": 1, "pageSize": 20, "totalPages": 10, "totalCount": 183 }
}
```

---

### POST /admin/impersonate/{userId}
Issue a JWT that acts as the specified user. Original admin identity recorded in audit log (I18).

**Auth required:** Yes — `is_admin = true`

**Request:** No body.

**Response 200:**
```json
{
  "data": {
    "accessToken": "eyJhbGci...",
    "tokenType": "Bearer",
    "expiresIn": 3600,
    "impersonating": {
      "userId": "uuid",
      "displayName": "Alice Chen",
      "email": "alice@example.com"
    },
    "originalAdmin": {
      "userId": "uuid",
      "displayName": "Super Admin"
    }
  }
}
```

**Errors:**
| Code | Condition |
|------|-----------|
| `FORBIDDEN` 403 | Not a platform admin |
| `NOT_FOUND` 404 | User does not exist |

---

### GET /admin/audit-log
Browse the immutable platform-wide audit log.

**Auth required:** Yes — `is_admin = true`

**Query params:** `?page=1&pageSize=50&eventSlug=hackathon-2026&actorId=uuid&action=SUBMIT_SCORE`

**Response 200:**
```json
{
  "data": [
    {
      "auditId": "uuid",
      "actorId": "uuid",
      "actorName": "Alice Chen",
      "action": "SUBMIT_SCORE",
      "resourceType": "assignment",
      "resourceId": "uuid",
      "eventSlug": "hackathon-2026",
      "ipHash": "sha256:abcdef...",
      "createdAt": "2026-10-22T14:33:00Z",
      "metadata": {
        "weightedScore": 8.275
      }
    }
  ],
  "meta": { "page": 1, "pageSize": 50, "totalPages": 4, "totalCount": 183 }
}
```

---

### GET /events/{slug}/audit-log
Browse the audit log scoped to one event. Available to organizer and admin.

**Auth required:** Yes — `organizer` or `is_admin = true`

**Query params:** `?page=1&pageSize=50&action=SUBMIT_SCORE`

**Response 200:** Same shape as platform audit log, filtered to this event.

---

## Module 8: Certificates

### GET /certificates/verify/{hash}
Verify a certificate by its verification hash (public, no auth).

**Auth required:** No

**Response 200:**
```json
{
  "data": {
    "certificateId": "uuid",
    "recipientName": "Alice Chen",
    "eventName": "Hackathon 2026",
    "rank": 1,
    "submissionTitle": "TerraBot",
    "issuedAt": "2026-10-29T10:00:00Z",
    "verificationHash": "abc123...",
    "downloadUrl": "http://localhost:9000/certificates/uuid.pdf"
  }
}
```

**Errors:**
| Code | Condition |
|------|-----------|
| `NOT_FOUND` 404 | Hash does not match any certificate |

---

## Complete Endpoint Index

| # | Method | Path | Auth | Module |
|---|--------|------|------|--------|
| 1 | POST | `/auth/register` | No | Auth |
| 2 | POST | `/auth/login` | No | Auth |
| 3 | POST | `/auth/refresh` | Cookie | Auth |
| 4 | POST | `/auth/logout` | JWT | Auth |
| 5 | POST | `/events` | JWT | Events |
| 6 | GET | `/events` | No | Events |
| 7 | GET | `/events/{slug}` | Optional | Events |
| 8 | PATCH | `/events/{slug}` | Organizer | Events |
| 9 | POST | `/events/{slug}/status` | Organizer | Events |
| 10 | POST | `/events/{slug}/rubric` | Organizer | Events |
| 11 | POST | `/events/{slug}/judges` | Organizer | Events |
| 12 | DELETE | `/events/{slug}/judges/{judgeId}` | Organizer | Events |
| 13 | POST | `/events/{slug}/register` | JWT | Registration |
| 14 | DELETE | `/events/{slug}/register` | JWT | Registration |
| 15 | POST | `/events/{slug}/teams` | Participant | Teams |
| 16 | GET | `/events/{slug}/teams/mine` | Participant | Teams |
| 17 | POST | `/teams/join` | Participant | Teams |
| 18 | DELETE | `/teams/{teamId}/members/me` | Member | Teams |
| 19 | POST | `/events/{slug}/submissions` | Team Owner | Submissions |
| 20 | PATCH | `/submissions/{id}` | Team Owner | Submissions |
| 21 | POST | `/submissions/{id}/submit` | Team Owner | Submissions |
| 22 | POST | `/submissions/{id}/files` | Team Owner | Submissions |
| 23 | GET | `/events/{slug}/submissions` | Optional | Submissions |
| 24 | POST | `/submissions/{id}/disqualify` | Organizer | Submissions |
| 25 | GET | `/judge/queue` | Judge | Judging |
| 26 | GET | `/judge/assignments/{id}` | Judge (own) | Judging |
| 27 | POST | `/judge/assignments/{id}/scores` | Judge (own) | Judging |
| 28 | POST | `/judge/assignments/{id}/recuse` | Judge (own) | Judging |
| 29 | POST | `/events/{slug}/normalize` | Organizer | Judging |
| 30 | POST | `/events/{slug}/votes` | Participant | Voting |
| 31 | GET | `/events/{slug}/results` | No | Voting |
| 32 | GET | `/admin/users` | Admin | Admin |
| 33 | POST | `/admin/impersonate/{userId}` | Admin | Admin |
| 34 | GET | `/admin/audit-log` | Admin | Admin |
| 35 | GET | `/events/{slug}/audit-log` | Organizer/Admin | Admin |
| 36 | GET | `/certificates/verify/{hash}` | No | Certificates |
