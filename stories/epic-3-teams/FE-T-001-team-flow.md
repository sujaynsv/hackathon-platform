---
id: FE-T-001
title: Team Registration + Team Management Flow
epic: teams
owner: Keerthika (frontend)
status: "[ ] not-started"
branch: story/FE-T-001-team-flow
blocks: FE-S-001
blocked-by: FE-E-001, T-003
---

# FE-T-001 · Team Registration + Team Management Flow

## Context (Read ALL of these before writing any code)
- `web/src/types/api.ts` — `Team`, `TeamMember` types
- `web/src/lib/api.ts` — `apiClient.registerForEvent()`, `apiClient.createTeam()`, `apiClient.joinTeam()`, `apiClient.getMyTeam()`

## What We're Building

The complete participant journey for joining an event and getting into a team. Participants register for an event, then either create a team (get invite code) or join an existing team with an invite code. The team dashboard shows members and the shared invite code.

**UI sections** (all under `/events/[slug]/participate`):
1. **Register step** — "Register for this Event" CTA (only shown if not yet registered)
2. **Team step** — shown after registration:
   - If no team: "Create Team" form OR "Join with Code" input
   - If on team: Team card with members list + invite code + leave button

## Files to Create

### web/src/app/events/[slug]/participate/page.tsx
Multi-step flow:
```
Step 1: Event Registration (POST /events/{slug}/register)
  → Shows if user is not yet registered
  → "Register" button → 201 → advance to step 2

Step 2: Team Setup
  → GET /events/{slug}/teams/mine first
  → If has team: show team card (step 3)
  → If no team: show two options:
    a) Create Team (name input → POST /events/{slug}/teams)
    b) Join with Code (8-char input → POST /teams/join)

Step 3: Team Dashboard
  → Show team name, invite code (copyable), members with roles
  → Leave team button (with confirmation)
```

### web/src/components/teams/TeamCard.tsx
```typescript
interface TeamCardProps {
  team: Team;
  onLeave: () => void;
}
// Shows:
// - Team name (large)
// - Invite code: copyable pill "HACK2026" with copy-to-clipboard button
// - Members list: avatar + name + role badge (Leader / Member)
// - "Leave Team" button (hidden for leader)
```

### web/src/components/teams/CreateTeamForm.tsx
Simple form: team name input + submit. Shows 409 error "Team name taken" inline.

### web/src/components/teams/JoinWithCodeForm.tsx
```typescript
// 8-character uppercase input with auto-uppercase transform
// Validates: exactly 8 alphanumeric chars before submit
// Shows errors: "Invalid code" (404), "Team is full" (422)
```

### web/src/hooks/useMyTeam.ts
```typescript
export function useMyTeam(eventSlug: string) {
  return useQuery({
    queryKey: ['myTeam', eventSlug],
    queryFn: () => apiClient.getMyTeam(eventSlug),
  });
}
```

## Design Requirements
- Step indicator at top (1. Register → 2. Join Team → 3. Ready!)
- Team card: gradient border, invite code in monospace font
- Copy button: shows "Copied!" for 2 seconds after click
- Members: avatar circles (initials fallback), role badge
- Leave team: requires typing "LEAVE" in confirmation input

## Definition of Done
- [ ] Register CTA calls API and advances to team step
- [ ] Create team form → 201 → shows team card with invite code
- [ ] Join with code → 201 → shows team card
- [ ] Invite code copy button works
- [ ] Leave team (non-leader) → confirmation → API call → resets to step 2
- [ ] Error states shown inline (team full, code not found, etc.)
- [ ] `npm run build` → 0 TypeScript errors
