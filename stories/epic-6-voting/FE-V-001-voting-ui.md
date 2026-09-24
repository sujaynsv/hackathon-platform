---
id: FE-V-001
title: Voting UI + Results Leaderboard
epic: voting
owner: Keerthika (frontend)
status: "[ ] not-started"
branch: story/FE-V-001-voting-ui
blocks: none (last frontend story)
blocked-by: FE-S-001, V-001, V-002
---

# FE-V-001 · Voting UI + Results Leaderboard

## Context (Read ALL of these before writing any code)
- `web/src/types/api.ts` — `Vote`, `VoteResult` types
- `web/src/lib/api.ts` — `apiClient.castVote()`, `apiClient.retractVote()`, `apiClient.getVoteResults()`

## What We're Building

Public voting on submissions during the voting window, and a ranked leaderboard after voting closes. Any authenticated user can vote once per submission. The UI shows real-time vote counts (cached) and disables voting for own team or after already voted. The leaderboard shows both vote rank and judge rank side by side.

**Pages:**
- `/events/[slug]/vote` — voting interface (visible only during voting window)
- `/events/[slug]/results` — leaderboard (visible after voting closes)

## Files to Create

### web/src/app/events/[slug]/vote/page.tsx
```
- Only renders if event status = 'voting' AND within voting window
- If status is wrong → redirect to /events/[slug] with a status message
- Shows VoteCard grid for all submissions
- User's votes tracked in local state (optimistic update)
```

### web/src/app/events/[slug]/results/page.tsx
```
- Available after voting_closes_at
- Ranked leaderboard table + trophy icons for top 3
- Shows voteRank and judgeRank side by side for each submission
- Tab: "Overall" | per-track tabs
- "Certificate" button per row (links to verification URL if issued)
```

### web/src/components/voting/VoteCard.tsx
```typescript
interface VoteCardProps {
  submission: SubmissionSummary;
  hasVoted: boolean;
  isOwnTeam: boolean;
  voteCount: number;
  onVote: () => void;
  onRetract: () => void;
}
// Shows: cover image, title, team name, vote count, vote button
// Vote button states:
//   - Normal: "♥ Vote" (indigo outlined)
//   - Voted: "♥ Voted" (indigo filled, click to retract)
//   - Own team: "Your Submission" (disabled, gray)
//   - Loading: spinner
// Optimistic update: update count immediately, revert on API error
```

### web/src/components/voting/VotingTimer.tsx
```typescript
// Countdown to voting close: "Voting closes in 3 hours 42 minutes"
// Turns red when < 1 hour remains
// Auto-refreshes submission list when voting closes (redirects to results)
```

### web/src/components/voting/Leaderboard.tsx
```typescript
interface LeaderboardProps {
  results: VoteResult[];
  trackId?: string; // for per-track view
}
// Rank column: 🥇 🥈 🥉 for top 3, number for rest
// Vote Rank and Judge Rank columns side by side
// Medal animation for rank 1 (CSS keyframes)
// Filter by track via tabs
```

### web/src/hooks/useVotingResults.ts
```typescript
export function useVotingResults(eventSlug: string, trackId?: string) {
  return useQuery({
    queryKey: ['votingResults', eventSlug, trackId],
    queryFn: () => apiClient.getVoteResults(eventSlug, trackId),
    // Only fetch if voting has closed
    staleTime: 60_000, // results cached for 60s (matches server Redis TTL)
  });
}
```

## Design Requirements
- Voting cards: grid 3-col desktop, 1-col mobile; heart icon for vote button
- Voted state: filled heart, count increments optimistically
- Leaderboard: striped rows, trophy emoji, confetti animation on first load (CSS)
- Gold/silver/bronze colors for top 3 ranks
- Smooth count increment animation (CSS counter animation or framer-motion)

## Definition of Done
- [ ] `/events/[slug]/vote` shows submissions with vote buttons only during voting window
- [ ] Voting outside window → redirected with message
- [ ] Cast vote → optimistic update → confirmed on API response
- [ ] Retract vote → count decrements
- [ ] Own team submission → "Your Submission" disabled
- [ ] `/events/[slug]/results` shows leaderboard ranked by vote count
- [ ] Top 3 have trophy icons (🥇🥈🥉)
- [ ] Track tab filter works
- [ ] `npm run build` → 0 TypeScript errors
