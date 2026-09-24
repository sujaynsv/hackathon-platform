# Wireframing Plan — Dogfood Hackathon Platform

## Research Summary: What Tool to Use

### Options Evaluated

| Tool | Fit for this project | Verdict |
|------|---------------------|---------|
| **Excalidraw MCP** (`mcp-excalidraw-server`) | Already installed. Works by placing JSON elements on canvas. The CLI approach fails for complex layouts because argument length limits prevent sending full page definitions in one command. The right workflow is: write `.excalidraw` JSON files → import them. | Use — but via file import, not inline CLI args |
| **Figma MCP** | Best fidelity. Requires paid Figma plan + Dev seat + desktop app. First-party from Figma. Ideal for production design handoff. | Not immediately usable — requires Figma plan |
| **HTML static mockups** | AI generates real browser-rendered HTML. Can open directly in browser. No tool dependency. Version-controlled alongside code. Matches the actual tech stack (Next.js). Can later be evolved into real components. The gap from wireframe → production is nearly zero. | Best fit for this team |
| **Whimsical MCP** | Good for flowcharts and user flows. Requires Whimsical account + token. Better for diagram-style flows than page wireframes. | Not needed — diagrams already done in Mermaid |

---

## Decision: HTML Wireframes (Best Approach for This Project)

**Reasons:**
1. **Zero tool dependency** — every browser can render them, no sign-ups needed
2. **Our design tokens are already written** in `docs/frontend-standards.md` — we can apply them directly in the wireframe HTML
3. **No translation gap** — person-c building the Next.js frontend can open the wireframe and extract styles directly
4. **AI is excellent at HTML** — Antigravity can generate highly accurate, pixel-faithful wireframes using the design system tokens
5. **Version-controlled** — lives in `docs/wireframes/` alongside all other project docs
6. **Interactive** — can show hover states, active states, disabled states in CSS

**Excalidraw use:** Still useful for **flow diagrams** (user journeys, navigation maps) — not for detailed page wireframes. The Mermaid diagrams in `docs/diagrams.md` already cover this.

---

## What to Wireframe

Based on the frontend stories and the user flows in `docs/diagrams.md §6`, these are the 12 pages that need wireframes:

| # | Page | Route | Priority | Owner story |
|---|------|--------|----------|-------------|
| 1 | Events Listing | `/events` | HIGH | FE-E-001 |
| 2 | Event Detail | `/events/[slug]` | HIGH | FE-E-001 |
| 3 | Login | `/login` | HIGH | FE-A-001 |
| 4 | Register | `/register` | HIGH | FE-A-001 |
| 5 | Team Setup Flow | `/events/[slug]/participate` | HIGH | FE-T-001 |
| 6 | Team Dashboard | `/events/[slug]/team` | HIGH | FE-T-001 |
| 7 | Submission Editor | `/events/[slug]/submit` | HIGH | FE-S-001 |
| 8 | Submission Gallery | `/events/[slug]/submissions` | HIGH | FE-S-001 |
| 9 | Organizer Dashboard | `/events/[slug]/manage` | MEDIUM | FE-E-002 |
| 10 | Judge Queue | `/judging` | MEDIUM | FE-J-001 |
| 11 | Judge Scoring | `/judging/assignments/[id]` | MEDIUM | FE-J-001 |
| 12 | Voting + Leaderboard | `/events/[slug]/results` | MEDIUM | FE-V-001 |

---

## Wireframe Specification per Page

Each HTML wireframe will contain:
- Full page layout at 1200px desktop
- Dark theme using exact design tokens from `docs/frontend-standards.md`
- Real navigation bar (not placeholder boxes)
- Annotated sections showing: API call that populates this section, relevant invariant, empty/loading state behavior
- Interactive CSS: hover states on buttons and cards, focus rings on inputs
- Mobile breakpoint note at bottom (not a full mobile layout — just annotation)

---

## File Structure

```
docs/wireframes/
├── index.html              ← Wireframe navigation hub (links to all 12 pages)
├── _shared.css             ← Design tokens from frontend-standards.md (shared across all wireframes)
├── 01-events-listing.html
├── 02-event-detail.html
├── 03-login.html
├── 04-register.html
├── 05-team-setup.html
├── 06-team-dashboard.html
├── 07-submission-editor.html
├── 08-submission-gallery.html
├── 09-organizer-dashboard.html
├── 10-judge-queue.html
├── 11-judge-scoring.html
└── 12-voting-leaderboard.html
```

---

## Excalidraw — What We Use It For

The Excalidraw MCP (`mcp-excalidraw-server`) is best used for **navigation flow maps** — not detailed page layouts. We will generate two flow diagrams in Excalidraw:

1. **User Journey Map** — shows how a participant moves through the app (register → team → submit → vote)
2. **Role-Based Navigation Map** — shows which pages each role (public, participant, judge, organizer, admin) can access

These will be exported as `.excalidraw` files to `docs/wireframes/flows/`.

---

## Implementation Order

```
Step 1: Create _shared.css with all design tokens
Step 2: Create index.html hub
Step 3: Pages 03 (Login) + 04 (Register) — simplest, establish the pattern
Step 4: Pages 01 (Events Listing) + 02 (Event Detail) — core public pages
Step 5: Pages 05 (Team Setup) + 06 (Team Dashboard) — participant flow
Step 6: Pages 07 (Submission Editor) + 08 (Submission Gallery)
Step 7: Pages 09 (Organizer) + 10-11 (Judge)
Step 8: Page 12 (Voting + Leaderboard)
Step 9: Excalidraw flow diagrams via file import
```
