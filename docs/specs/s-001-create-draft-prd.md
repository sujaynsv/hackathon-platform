# S-001: Create Submission Draft - Product Requirements Document

## What We're Building
We are building the ability for a team to create a draft submission for an event in the hackathon platform. A submission represents the project a team has built during the event.

## Why We're Building It
To allow participants to submit their projects for judging. Submissions must start in a `draft` state so that teams can work on them incrementally before finalizing them.

## Requirements & Business Rules
1. **Event State**: Submissions can only be created if the event is currently in the `submissions_open` status (Invariant I9).
2. **Team Constraint**: Only members of a team registered for the event can create a submission. The team is inferred from the caller's identity (the user making the request).
3. **Uniqueness**: A team can only have **one** submission per event (Invariant I2). Creating a second submission will fail.
4. **Draft State**: All new submissions start in the `draft` status.
5. **Track Assignment**: Teams can optionally assign their submission to a specific track. If provided, the track ID must belong to the event.

## Success Criteria
- User can successfully create a draft submission via `POST /api/v1/events/{slug}/submissions`.
- Trying to submit when not part of a team returns `403 FORBIDDEN`.
- Trying to submit multiple times for the same event returns `409 DUPLICATE_RESOURCE`.
- Trying to submit when the event is not accepting submissions returns `422 INVARIANT_VIOLATION`.
- Trying to assign to an invalid track returns `400 VALIDATION_ERROR`.
