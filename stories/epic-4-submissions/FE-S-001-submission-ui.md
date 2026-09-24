---
id: FE-S-001
title: Submission Editor + Gallery
epic: submissions
owner: Keerthika (frontend)
status: "[ ] not-started"
branch: story/FE-S-001-submission-ui
blocks: FE-J-001, FE-V-001
blocked-by: FE-T-001, S-003
---

# FE-S-001 · Submission Editor + Gallery

## Context (Read ALL of these before writing any code)
- `web/src/types/api.ts` — `Submission`, `SubmissionStatus` types
- `web/src/lib/api.ts` — `apiClient.createSubmission()`, `apiClient.updateSubmission()`, `apiClient.submitSubmission()`, `apiClient.uploadFile()`, `apiClient.listSubmissions()`

## What We're Building

The submission workspace for team members and the public gallery of all submissions. Team members can create/edit their submission, upload a cover image, and finalize with the "Submit" action. The gallery is public-facing and sortable.

**Pages:**
- `/events/[slug]/submit` — submission workspace (team-member only)
- `/events/[slug]/submissions` — public gallery

## Files to Create

### web/src/app/events/[slug]/submit/page.tsx
Protected workspace (must be team member):
```
Step 1: Create draft (if no submission yet) — form with title, description, trackId, repoUrl, demoUrl
Step 2: Edit draft — same form pre-filled, editable
Step 3: Upload cover — drag-and-drop image upload, preview
Step 4: Review + Submit — final check, "Submit Project" button with confirmation

Status bar: "Draft" | "Submitted" badge at top
Deadline countdown: "Submission closes in 2 days 4 hours"
```

### web/src/app/events/[slug]/submissions/page.tsx
Public gallery:
- Filter bar: track selector dropdown + search by title
- Sort options: "Most Voted" (after voting) | "Top Scored" (after judging) | "Newest"
- Grid of SubmissionCard components with infinite scroll OR pagination
- Loading skeleton cards

### web/src/components/submissions/SubmissionCard.tsx
```typescript
// Cover image (or gradient fallback), title, team name,
// track badge, vote count (if visible), rank (if available), "View" button
```

### web/src/components/submissions/CoverUpload.tsx
```typescript
// Drag-and-drop zone:
// - Dotted border → solid on drag-over
// - Preview of uploaded image
// - File type validation (image only) client-side before API call
// - Progress bar during upload (XHR with progress event)
// - Error: "Too large" (>50MB), "Invalid type"
// - Replace button to swap existing cover
```

### web/src/components/submissions/SubmissionForm.tsx
```typescript
interface SubmissionFormProps {
  initialData?: Partial<Submission>;
  tracks: Track[];
  onSave: (data: SubmissionFormData) => Promise<void>;
}
// Fields: title (required), description (textarea, optional),
//   trackId (select, optional), repoUrl (url input), demoUrl (url input)
// Auto-save: debounced PATCH every 2 seconds after user stops typing
```

### web/src/hooks/useSubmission.ts
```typescript
export function useMySubmission(eventSlug: string) {
  return useQuery({
    queryKey: ['mySubmission', eventSlug],
    queryFn: () => apiClient.getMySubmission(eventSlug),
  });
}
```

## Design Requirements
- Submission workspace: two-column layout (form left, preview right)
- Cover upload zone: large, prominent, with animated dashed border on hover
- Deadline countdown: red pulsing dot when < 24 hours
- "Submit" button: requires explicit click of "I confirm this is my final submission"
- Gallery: Masonry grid or uniform card grid with hover effects
- Submitted status: green checkmark badge on submission card for own team

## Definition of Done
- [ ] `/events/[slug]/submit` creates draft, allows edits, uploads cover
- [ ] Auto-save working (no manual save button for draft)
- [ ] "Submit" confirmation dialog → API call → status badge changes to "Submitted"
- [ ] `/events/[slug]/submissions` public gallery loads correctly
- [ ] Track filter, sort options work
- [ ] Cover upload with drag-and-drop + preview
- [ ] `npm run build` → 0 TypeScript errors
