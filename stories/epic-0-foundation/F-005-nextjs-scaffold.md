---
id: F-005
title: Next.js 14 Scaffold + API Client + TypeScript Types
epic: foundation
owner: Keerthika (frontend)
status: "[ ] not-started"
branch: story/F-005-nextjs-scaffold
blocks: FE-A-001
blocked-by: F-001
---

# F-005 · Next.js 14 Scaffold + API Client + TypeScript Types

## Context (Read ALL of these before writing any code)
- `docs/api-design.md` — every endpoint's request/response contract (TypeScript types mirror this exactly)
- `docs/architecture.md §3.2` — standard `ApiResponse<T>` envelope and `ApiErrorResponse` shape
- `MASTER-CONTEXT.md` — confirmed frontend stack: Next.js 14 App Router, TanStack Query v5, Tailwind CSS, TypeScript strict mode

## What to Build

### Next.js 14 App Router project
Initialize in the `web/` directory with:
- TypeScript strict mode
- App Router (not Pages Router)
- Tailwind CSS
- ESLint

```bash
cd web
npx create-next-app@14 . --typescript --tailwind --eslint --app --src-dir --import-alias "@/*"
```

### web/src/types/api.ts — All API TypeScript types
This is the contract between frontend and backend. Mirrors `docs/api-design.md` exactly.

```typescript
// ─── Standard Envelope ─────────────────────────────────────────────────────
export interface ApiMeta {
  requestId: string;
  timestamp: string;
  page?: number;
  pageSize?: number;
  totalCount?: number;
  totalPages?: number;
}

export interface ApiResponse<T> {
  data: T;
  meta: ApiMeta;
}

export interface ApiError {
  code: string;
  message: string;
  details?: Record<string, unknown>;
}

export interface ApiErrorResponse {
  error: ApiError;
  meta: ApiMeta;
}

// ─── Auth ───────────────────────────────────────────────────────────────────
export interface RegisterRequest {
  email: string;
  password: string;
  displayName: string;
}

export interface RegisterResponse {
  user: UserProfile;
  accessToken: string;
  refreshToken: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  user: UserProfile;
  accessToken: string;
  refreshToken: string;
}

export interface UserProfile {
  id: string;
  email: string;
  displayName: string;
  avatarUrl: string | null;
  isAdmin: boolean;
  createdAt: string;
}

// ─── Events ─────────────────────────────────────────────────────────────────
export type EventStatus =
  | 'draft'
  | 'registration_open'
  | 'submissions_open'
  | 'judging'
  | 'voting'
  | 'results_published'
  | 'archived';

export interface Event {
  id: string;
  slug: string;
  title: string;
  description: string | null;
  bannerUrl: string | null;
  status: EventStatus;
  organizerId: string;
  registrationOpensAt: string | null;
  registrationClosesAt: string | null;
  submissionDeadlineAt: string | null;
  judgingDeadlineAt: string | null;
  votingOpensAt: string | null;
  votingClosesAt: string | null;
  maxTeamSize: number;
  createdAt: string;
}

// ─── Teams ──────────────────────────────────────────────────────────────────
export interface Team {
  id: string;
  eventId: string;
  name: string;
  inviteCode: string;
  members: TeamMember[];
  createdAt: string;
}

export interface TeamMember {
  userId: string;
  displayName: string;
  avatarUrl: string | null;
  role: 'leader' | 'member';
  joinedAt: string;
}

// ─── Submissions ─────────────────────────────────────────────────────────────
export type SubmissionStatus = 'draft' | 'submitted' | 'disqualified';

export interface Submission {
  id: string;
  teamId: string;
  eventId: string;
  trackId: string | null;
  title: string;
  description: string | null;
  repoUrl: string | null;
  demoUrl: string | null;
  coverUrl: string | null;
  status: SubmissionStatus;
  finalScore: number | null;
  overallRank: number | null;
  trackRank: number | null;
  submittedAt: string | null;
  createdAt: string;
}

// ─── Judging ─────────────────────────────────────────────────────────────────
export interface JudgeAssignment {
  id: string;
  submissionId: string;
  submission: Submission;
  rubricId: string;
  status: 'pending' | 'in_progress' | 'completed' | 'recused';
  assignedAt: string;
  completedAt: string | null;
}

export interface ScoreSubmission {
  criterionId: string;
  rawScore: number;
  notes?: string;
}

// ─── Voting ───────────────────────────────────────────────────────────────────
export interface VoteRequest {
  submissionId: string;
}

export interface VoteResponse {
  voted: boolean;
}
```

### web/src/lib/api.ts — Typed API client

```typescript
const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:80/api/v1';

export class ApiClient {
  private token: string | null = null;

  setToken(token: string) { this.token = token; }
  clearToken() { this.token = null; }

  private async request<T>(
    method: string,
    path: string,
    body?: unknown,
    formData?: FormData
  ): Promise<T> {
    const headers: Record<string, string> = {};
    if (this.token) headers['Authorization'] = `Bearer ${this.token}`;
    if (body) headers['Content-Type'] = 'application/json';

    const res = await fetch(`${API_BASE}${path}`, {
      method,
      headers,
      body: formData ?? (body ? JSON.stringify(body) : undefined),
    });

    const json = await res.json();

    if (!res.ok) {
      const err = json as ApiErrorResponse;
      throw new ApiClientError(err.error.code, err.error.message, res.status);
    }

    return (json as ApiResponse<T>).data;
  }

  // Auth
  register(req: RegisterRequest) {
    return this.request<RegisterResponse>('POST', '/auth/register', req);
  }
  login(req: LoginRequest) {
    return this.request<LoginResponse>('POST', '/auth/login', req);
  }
  logout() {
    return this.request<void>('POST', '/auth/logout');
  }

  // Events
  listEvents() {
    return this.request<Event[]>('GET', '/events');
  }
  getEvent(slug: string) {
    return this.request<Event>('GET', `/events/${slug}`);
  }

  // Add more methods as stories are completed...
}

export class ApiClientError extends Error {
  constructor(
    public readonly code: string,
    message: string,
    public readonly status: number
  ) {
    super(message);
    this.name = 'ApiClientError';
  }
}

export const apiClient = new ApiClient();
```

### web/src/app/layout.tsx — Root layout with TanStack Query
```typescript
'use client';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Inter } from 'next/font/google';
import './globals.css';

const inter = Inter({ subsets: ['latin'] });
const queryClient = new QueryClient();

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body className={inter.className}>
        <QueryClientProvider client={queryClient}>
          {children}
        </QueryClientProvider>
      </body>
    </html>
  );
}
```

### web/Dockerfile
```dockerfile
FROM node:20-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

FROM node:20-alpine AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

FROM node:20-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
COPY --from=builder /app/public ./public
EXPOSE 3000
CMD ["node", "server.js"]
```

## Tests Required
- `npm run build` — zero TypeScript errors (strict mode)
- `npm run lint` — zero ESLint warnings
- Type-level test: `ApiResponse<UserProfile>` compiles against the `data + meta` shape
- Smoke test: dev server starts `npm run dev`, `GET http://localhost:3000` returns 200

## Definition of Done
- [ ] `npm install` → no errors
- [ ] `npm run build` → 0 TypeScript errors
- [ ] `npm run lint` → 0 warnings
- [ ] `web/src/types/api.ts` — all types for all 8 epics defined
- [ ] `web/src/lib/api.ts` — typed API client with auth methods
- [ ] `web/Dockerfile` multi-stage build works
- [ ] `docker compose up web` starts Next.js and serves the landing page
