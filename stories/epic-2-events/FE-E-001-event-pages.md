---
id: FE-E-001
title: Event Listing + Event Detail Pages
epic: events
owner: Keerthika (frontend)
status: "[ ] not-started"
branch: story/FE-E-001-event-pages
blocks: FE-T-001
blocked-by: FE-A-001, E-001
---

# FE-E-001 · Event Listing + Event Detail Pages

## Context (Read ALL of these before writing any code)
- `web/src/types/api.ts` — `Event`, `EventStatus`, `ApiResponse` types
- `web/src/lib/api.ts` — `apiClient.listEvents()`, `apiClient.getEvent(slug)`
- `docs/api-design.md §Events` — exact response shapes for GET /events and GET /events/{slug}

## What We're Building

The public-facing event browsing experience. A landing page shows all active events. Clicking an event shows its detail page with tracks, registration status, and dates. Status badges show the current lifecycle stage. Authenticated users see their role (participant, judge, organizer).

**Pages:**
- `/events` — events listing grid
- `/events/[slug]` — event detail with tracks, dates, registration CTA

## Files to Create

### web/src/app/events/page.tsx
Event listing using TanStack Query:
```typescript
const { data, isLoading } = useQuery({
  queryKey: ['events'],
  queryFn: () => apiClient.listEvents(),
});
```
**UI layout:**
- Grid of EventCard components (3 columns on desktop, 1 on mobile)
- Search/filter bar (client-side filter by title)
- Status filter tabs: All | Open | Judging | Voting | Closed
- Loading: 6 skeleton cards
- Empty state: illustration + "No events yet"

### web/src/app/events/[slug]/page.tsx
Event detail page:
- Hero banner (bannerUrl image or gradient fallback)
- Event title, description, status badge
- Key dates section (registration, submission, voting windows)
- Tracks list with descriptions
- Sidebar: registration CTA (if status=registration_open and user not registered)
- User's role badge if authenticated ("You are a Participant", "You are a Judge")
- Stats: number of participants, teams, submissions (if available)

### web/src/components/events/EventCard.tsx
```typescript
interface EventCardProps {
  event: Event;
}
// Shows: banner/gradient, title, status badge, registration deadline, track count
```

### web/src/components/events/StatusBadge.tsx
```typescript
// Maps EventStatus to colored badge:
// draft             → gray "Draft"
// registration_open → green "Open for Registration"
// submissions_open  → blue "Accepting Submissions"
// judging           → purple "Judging in Progress"
// voting            → yellow "Public Voting"
// results_published → indigo "Results Published"
// archived          → gray "Archived"
```

### web/src/components/events/DateTimeline.tsx
Visual timeline component showing all event milestones with past/current/future indicators.

## Design Requirements
- Events grid: cards with hover lift effect (`transform hover:-translate-y-1 transition-all`)
- Banner: 16:9 aspect ratio, object-cover, with gradient overlay for text readability
- Status badges: pill-shaped, color-coded (green=open, blue=submissions, purple=judging, yellow=voting)
- Detail page hero: full-width banner with overlay gradient, title in white
- Dark mode throughout
- Date/time shown in user's local timezone (use `Intl.DateTimeFormat`)

## Definition of Done
- [ ] `/events` → lists all non-draft events from API
- [ ] Loading state shows skeleton cards
- [ ] Status filter tabs work (client-side)
- [ ] `/events/[slug]` shows event detail, tracks, dates
- [ ] Status badge correct color/label for all 7 statuses
- [ ] Authenticated user's role shown on detail page
- [ ] `npm run build` → 0 TypeScript errors
