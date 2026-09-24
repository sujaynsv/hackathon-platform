---
id: FE-J-001
title: Judge Interface — Queue + Scoring Form
epic: judging
owner: Keerthika (frontend)
status: "[ ] not-started"
branch: story/FE-J-001-judge-ui
blocks: FE-V-001
blocked-by: FE-E-002, J-002
---

# FE-J-001 · Judge Interface — Queue + Scoring Form

## Context (Read ALL of these before writing any code)
- `web/src/types/api.ts` — `JudgeAssignment`, `AssignmentStatus`, `Score` types
- `web/src/lib/api.ts` — `apiClient.getJudgeQueue()`, `apiClient.getAssignmentDetail()`, `apiClient.submitScores()`, `apiClient.recuseAssignment()`

## What We're Building

A dedicated judge interface. Judges see a queue of submissions assigned to them, click into one to view the submission details, and score each criterion from the rubric. A visual scoring form with sliders and notes fields. Clear status tracking (pending, in_progress, completed, recused).

**Pages:**
- `/judging` — judge queue dashboard
- `/judging/assignments/[id]` — single assignment scoring interface

## Files to Create

### web/src/app/judging/page.tsx
Protected (judge only — show message "You are not assigned as a judge" if not in queue):
- Status filter tabs: Pending | In Progress | Completed | Recused | All
- List of AssignmentCard components
- Empty state: "All done! 🎉" when no pending assignments
- Progress bar: "12 of 15 scored"

### web/src/app/judging/assignments/[id]/page.tsx
Three-panel layout:
```
Left panel: Submission details
  - Cover image, title, team name, track
  - Description (expandable)
  - Repo URL + Demo URL links (open in new tab)

Right panel: Scoring form
  - Rubric criteria list
  - For each criterion:
    - Name + description
    - Score slider: 1 to maxScore (labeled tick marks)
    - Score input (synced with slider)
    - Notes textarea (optional)
    - Criterion weight displayed as percentage
  - Running weighted score preview: "Estimated score: 7.8 / 10"
  - "Submit Scores" button (disabled until all criteria scored)
  - "Recuse from this Assignment" link (opens modal with reason input)

Bottom bar: Assignment status + deadline countdown
```

### web/src/components/judging/ScoreCriterion.tsx
```typescript
interface ScoreCriterionProps {
  criterion: RubricCriterion;
  value: number;
  notes: string;
  onChange: (score: number, notes: string) => void;
}
// Slider: range input 1..maxScore, number input synced, notes textarea
// Visual: shows weight as "30% of total score"
```

### web/src/components/judging/AssignmentCard.tsx
```typescript
// In the queue list:
// - Submission title + team name
// - Status badge (Pending / In Progress / Completed / Recused)
// - Track badge
// - "Score Now" button → /judging/assignments/{id}
```

### web/src/components/judging/RecuseModal.tsx
```typescript
// Modal with:
// - Warning: "This cannot be undone. You will be removed from scoring this submission."
// - Reason textarea (required)
// - "Confirm Recusal" button
```

### web/src/hooks/useJudgeQueue.ts
```typescript
export function useJudgeQueue(status?: string) {
  return useQuery({
    queryKey: ['judgeQueue', status],
    queryFn: () => apiClient.getJudgeQueue({ status }),
    refetchInterval: 30_000, // refresh every 30s to catch new assignments
  });
}
```

## Design Requirements
- Queue: clean list with status-colored left border (yellow=pending, blue=in_progress, green=completed)
- Scoring form: slider styled in indigo, smooth transition
- Weighted score preview: animated number that updates as scores change
- All-scored indicator: green checkmark appears per criterion when scored
- Mobile: single-column layout, submission details collapsible

## Definition of Done
- [ ] `/judging` shows queue with correct status counts
- [ ] Status filter tabs work
- [ ] `/judging/assignments/[id]` shows submission + rubric criteria
- [ ] Scoring form: all criteria must be scored before submit enabled
- [ ] Submit scores → success → assignment status changes to "Completed" in UI
- [ ] Recuse modal → reason → API call → assignment disappears from pending queue
- [ ] Deadline countdown visible in scoring interface
- [ ] `npm run build` → 0 TypeScript errors
