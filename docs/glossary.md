# Glossary — Dogfood Platform Domain Vocabulary

> Shared language for the entire team. Every document, API, and variable name should use these exact terms.
> If you call something by a different name, update this glossary.

---

| Term | Definition |
|------|-----------|
| **Event** | A hackathon. The top-level container for everything else (tracks, teams, submissions, judges, votes). |
| **Track** | A category within an event. Submissions are submitted to exactly one track. Examples: "Open Track", "AI Track". |
| **EventRegistration** | A record of a participant opting into an event. Required before creating or joining a team. |
| **Team** | A group of 1–4 participants collaborating on one submission. Teams are event-scoped. |
| **TeamMember** | A user's membership in a team, with a role of `owner` or `member`. |
| **Invite Code** | A unique random token generated when a team is created. Shared out-of-band to let others join. |
| **Submission** | The project entry created by a team. One submission per team per event. |
| **Draft** | A submission that has been created but not yet finalized. Editable, not visible in public gallery. |
| **Submit** | The action of finalizing a submission (status: `draft` → `submitted`). Irreversible before deadline. |
| **Submission Window** | The period between `submission_opens_at` and `submission_deadline_at`. Submissions can be created and edited only during this window. |
| **Rubric** | A scoring template with weighted criteria. Created by the organizer per event (or per track). |
| **Criterion** | One scoring dimension in a rubric. Has a name, max score, and weight. Example: "Innovation", max 10, weight 0.25. |
| **Weights** | Decimal values assigned to rubric criteria. Must sum to exactly 1.0 for the rubric to be valid. |
| **JudgeAssignment** | A pairing of one judge to one submission. The judge evaluates that submission using the rubric. |
| **Judging Window** | The period between `judging_opens_at` and `judging_deadline_at`. Judges can only submit/update scores during this period. |
| **Raw Score** | The score a judge enters for a criterion, before any normalization. Stored permanently. |
| **Normalized Score** | The Z-score-corrected version of a raw score. Computed post-judging. Stored alongside raw score (non-destructive). |
| **Weighted Raw Total** | SUM(raw_score × criterion.weight) for a judge's full evaluation of one submission. |
| **Weighted Normalized Total** | SUM(normalized_score × criterion.weight) for a judge's full normalized evaluation. |
| **Final Score** | AVG(weighted_normalized_total) across all judges for a submission. Used for ranking. |
| **Normalization** | Z-score transformation applied per judge to remove scoring bias (harshness/leniency). |
| **Judging Integrity** | The property that scores are fair, isolated, normalized, and auditable. A key judging criterion (25% of score). |
| **Publish** | The organizer action that makes results visible to the public. Cannot be undone. Requires normalization to be complete first. |
| **Gallery** | The public-facing page listing all submitted projects for an event. Searchable. Shows scores only after results are published. |
| **Community Voting** | Optional public voting on submissions (T3). One vote per user per submission. Counts hidden until voting window closes. |
| **Voting Window** | The period between `voting_opens_at` and `voting_closes_at`. Votes can only be cast during this period. |
| **Voting Weight** | The fraction (0–1) of the final score contributed by community votes vs. judge scores. Configurable per event. |
| **Vote Count** | The number of community votes a submission has received. Hidden during the voting window (I12). |
| **Rate Limit** | Maximum number of actions allowed per user per time window. Applied to voting to prevent abuse. |
| **Sybil Attack** | Creating many fake accounts to cast multiple votes. Detected via IP hash correlation. |
| **Ballot Stuffing** | A single user voting multiple times for the same project. Prevented by DB unique constraint (I5). |
| **Bandwagon Effect** | Users voting for already-popular projects because they see high vote counts. Prevented by hiding counts during voting (I12). |
| **Conflict of Interest** | A judge being assigned to evaluate a submission from their own team. Prevented at assignment time (I3). |
| **Recusal** | A judge voluntarily stepping back from an assignment due to a conflict. Triggers re-assignment by organizer. |
| **AuditLog** | An append-only record of every significant action on the platform. Cannot be altered by anyone. |
| **Invariant** | A business rule that must never be violated. Enforced at the DB or application layer, not just the UI. |
| **Fixture Data** | The seed dataset provided by Hackathon Raptors. Used for testing and normalization proof. |
| **Acceptance Suite** | Automated tests run by Hackathon Raptors against the platform to verify tier claims. |
| **Acceptance Report** | The output of the acceptance suite — included in the repo as `acceptance-report.txt`. |
| **Certificate** | A verifiable record of participation, winning, or judging service. Has a tamper-proof HMAC hash. |
| **Verification Hash** | HMAC-SHA256 of certificate fields. Allows public verification without a database lookup. |
| **T1 / T2 / T3 / T4** | Tier levels from the hackathon spec. T1 = Core (required), T2 = Judging, T3 = Public, T4 = Stretch. |
| **OpenAPI** | The API specification format used to document all endpoints (T4 bonus, +3 pts). |
| **Z-Score** | A statistical measure: how many standard deviations a value is from the mean. Used for normalization. |
| **μ (mu)** | Mean — the average of a set of values. Used in Z-score: μ_j = mean of judge j's raw scores. |
| **σ (sigma)** | Standard deviation — the spread of a set of values. Used in Z-score: σ_j = std dev of judge j's raw scores. |
| **Bradley-Terry** | A statistical model for pairwise comparisons. Used for pairwise mode (T4 bonus, +5 pts). |
| **JWT** | JSON Web Token — the authentication token format used for session management. No external auth provider. |
| **Slug** | A URL-friendly string identifier for an event. Example: `dogfood-2026`. |
