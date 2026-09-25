# S-001: Create Submission Draft - Technical Specification

## 1. Architecture (Hexagonal)
We will implement this inside the `submissions` module (`internal/submissions`), adhering strictly to the dependency rule (no external package dependencies in `domain`, etc.).

## 2. Domain Layer (`internal/submissions/domain/submission.go`)
- Create `SubmissionStatus` enum (`draft`, `submitted`, `disqualified`).
- Create `Submission` struct with fields: `ID`, `TeamID`, `EventID`, `TrackID`, `Title`, `Description`, `RepoURL`, `DemoURL`, `CoverURL`, `Status`, `FinalScore`, `OverallRank`, `TrackRank`, `SubmittedAt`, `CreatedAt`, `UpdatedAt`.
- Create factory function `NewDraftSubmission`.
- Create domain function `CheckCanCreate(eventStatus string)` to enforce Invariant I9 (`submissions_open`).

## 3. Port Layer (`internal/submissions/port/in.go` & `out.go`)
- **Inbound**: `CreateSubmissionUseCase` interface and `CreateSubmissionCommand`.
- **Outbound**:
  - `SubmissionRepository` (Save, FindByTeamAndEvent, etc.)
  - `EventReader` (FindSummaryBySlug)
  - `TeamReader` (FindByEventAndUser)
  - `TrackReader` (FindByEventID)

## 4. Use Case Layer (`internal/submissions/usecase/create.go`)
- Create `CreateSubmissionService` implementing the use case port.
- **Workflow**:
  1. Find event via slug.
  2. Enforce I9: Event status must be `submissions_open` using domain function.
  3. Look up team for the caller; error if not found (403).
  4. Enforce I2: Check if team already has a submission (409).
  5. Validate track ID if provided.
  6. Instantiate `Submission` entity via `NewDraftSubmission`.
  7. Save to repository.

## 5. API Endpoint
- **Method**: `POST /api/v1/events/{slug}/submissions`
- **Request Body**:
  ```json
  {
    "title": "string",
    "description": "string",
    "trackId": "uuid",
    "repoUrl": "string",
    "demoUrl": "string"
  }
  ```
- **Response**: 201 Created with the draft submission DTO wrapped in `{"data": {...}}`.

## 6. Testing Strategy
- **Domain Tests**: Test factory methods and state validations.
- **Use Case Tests**: Mock ports to test validation, success paths, and invariant errors (I2, I9).
- **Integration Tests**: Full flow using testcontainers (PostgreSQL).
