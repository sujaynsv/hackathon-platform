# Dogfood Platform — Documentation Index

> **Dogfood 2026 Hackathon** | Hackathon Raptors
> Build the platform that will judge you.

---

## Documents

| Document | Description |
|----------|-------------|
| [Product Requirements (PRD)](./product-requirements.md) | Full user stories, acceptance criteria, NFRs, tier checklists |
| [Architecture](./architecture.md) | System design, service topology, Go package structure, invariant enforcement |
| [Mental Model](./mental-model.md) | Actors, domain objects, invariants, glossary |
| [Actors & Roles](./actors.md) | Detailed role capabilities and restrictions |
| [Domain Objects](./domain-objects.md) | All entities with fields and relationships |
| [Invariants](./invariants.md) | Hard business rules enforced at DB/app layer |
| [Glossary](./glossary.md) | Shared domain vocabulary |

## State Machines

| Diagram | Description |
|---------|-------------|
| [Event State Machine](./state-machines/event.md) | Event lifecycle: draft → archived |
| [Submission State Machine](./state-machines/submission.md) | Submission: draft → submitted → disqualified |
| [Team State Machine](./state-machines/team.md) | Team: forming → locked |
| [Judge Assignment State Machine](./state-machines/judge-assignment.md) | Assignment: pending → completed |
| [Normalization State Machine](./state-machines/normalization.md) | Scores: raw → normalized → published |

## User Flows

| Flow | Description |
|------|-------------|
| [Flow 1 — Registration & Submission](./flows/01-registration-submission.md) | Participant registers → joins team → submits project |
| [Flow 2 — Event Setup](./flows/02-event-setup.md) | Organizer creates event → rubric → publishes |
| [Flow 3 — Judge Assignment](./flows/03-judge-assignment.md) | Organizer invites judges → algorithmic assignment |
| [Flow 4 — Judging](./flows/04-judging.md) | Judge evaluates assigned submissions |
| [Flow 5 — Normalization & Results](./flows/05-normalization-results.md) | Z-score normalization → publish results |
| [Flow 6 — Community Voting](./flows/06-community-voting.md) | Public voting with anti-abuse measures |
| [Flow 7 — Certificates](./flows/07-certificates.md) | Auto-generated verifiable certificates |

---

*Next document to be written: `api-design.md` (API Design)*
