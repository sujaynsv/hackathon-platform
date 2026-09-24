---
id: FE-E-002
title: Organizer Dashboard — Create Event + Manage Status
epic: events
owner: Keerthika (frontend)
status: "[ ] not-started"
branch: story/FE-E-002-organizer-ui
blocks: FE-J-001
blocked-by: FE-E-001, E-002, E-003, E-004
---

# FE-E-002 · Organizer Dashboard — Create Event + Manage Status

## Context (Read ALL of these before writing any code)
- `web/src/types/api.ts` — `Event`, `EventStatus` types
- `web/src/lib/api.ts` — `apiClient.createEvent()`, `apiClient.updateEvent()`, `apiClient.inviteJudge()`
- `docs/api-design.md §Events` — POST /events, PATCH /events/{slug}, POST /events/{slug}/judges

## What We're Building

A protected organizer dashboard where event creators manage their events. Includes: event creation form, status transition button (advance through lifecycle), date editing, and judge invitation. Only users with the `organizer` role for the event can access the management panel.

**Pages:**
- `/events/create` — create new event form (authenticated users)
- `/events/[slug]/manage` — organizer dashboard for a specific event

## Files to Create

### web/src/app/events/create/page.tsx
Multi-step form with validation:
- Step 1: Basic info (slug, title, description, maxTeamSize)
- Step 2: Timeline (all 6 date/time fields)
- Step 3: Review + submit
- Progress indicator at top
- `apiClient.createEvent()` on submit → redirect to `/events/{slug}/manage`

### web/src/app/events/[slug]/manage/page.tsx
Protected (ProtectedRoute + organizer role check):
- **Status panel:** Current status badge + "Advance to Next Stage" button (disabled when cannot transition)
- **Edit panel:** PATCH form for title, description, dates (shows which fields are locked based on status)
- **Judges panel:** Invite by email form + list of current judges with remove button
- **Rubric panel:** Create rubric + add criteria with weights (weight sum indicator showing total vs 1.0)
- **Stats panel:** Counts of registrations, teams, submissions

### web/src/components/organizer/StatusAdvancer.tsx
```typescript
// Shows current status and the next valid status (per state machine)
// nextStatus: draft→registration_open, registration_open→submissions_open, etc.
// Button: "Open Registration" | "Close Registration, Open Submissions" | etc.
// Calls apiClient.updateEvent({ status: nextStatus })
// Confirmation modal before advancing (cannot go back!)
```

### web/src/components/organizer/RubricBuilder.tsx
```typescript
// Visual rubric editor:
// - Add criterion button
// - Each criterion: name input, weight slider (0.01–1.00), maxScore input, description
// - Running total weight indicator: "Total: 0.70 / 1.00" (red if not 1.0, green if exactly 1.0)
// - Validate button: calls POST /rubric/validate → shows validation result
```

### web/src/components/organizer/JudgeInviter.tsx
```typescript
// Email input + "Invite" button
// Shows list of current judges with "Remove" option
// Error display: "Email not registered" (404), "Already a judge" (409)
```

## Design Requirements
- Multi-step form with animated step transitions
- Status advancement: confirmation dialog with warning "This cannot be undone"
- Weight slider: color-coded (red when total > 1.0 or < 0.9, yellow 0.9–0.99, green = 1.0)
- Organizer dashboard: sidebar navigation (Status | Edit | Judges | Rubric | Stats)

## Definition of Done
- [ ] `/events/create` → multi-step form → creates event → redirect to manage
- [ ] `/events/[slug]/manage` accessible only to event organizer
- [ ] Status advance button shows correct next step and requires confirmation
- [ ] Rubric builder enforces weight sum = 1.0 visually
- [ ] Judge invite works with API error display
- [ ] `npm run build` → 0 TypeScript errors
