---
id: FE-A-001
title: Auth Pages — Register, Login, Logout (Next.js)
epic: auth
owner: Keerthika (frontend)
status: "[ ] not-started"
branch: story/FE-A-001-auth-pages
blocks: FE-E-001
blocked-by: F-005, A-002
---

# FE-A-001 · Auth Pages — Register, Login, Logout

## Context (Read ALL of these before writing any code)
- `web/src/types/api.ts` — `RegisterRequest`, `LoginRequest`, `RegisterResponse`, `LoginResponse` types (from F-005)
- `web/src/lib/api.ts` — `apiClient.register()`, `apiClient.login()`, `apiClient.logout()` methods
- `docs/api-design.md §Auth` — exact API shapes to match

## What We're Building

Three auth pages plus a shared auth context that manages the logged-in state across the app. After login/register, JWT tokens are stored in memory (access token) and an HTTP-only cookie (refresh token is handled server-side via a Next.js route). The auth context provides `user`, `isAuthenticated`, `login()`, `logout()`, `register()` to all components.

**Pages to build:**
- `/login` — Login form
- `/register` — Registration form
- Logout (no page — action from nav dropdown)

## Files to Create

### web/src/context/AuthContext.tsx
```typescript
'use client';
import { createContext, useContext, useState, useCallback } from 'react';
import { apiClient } from '@/lib/api';
import type { UserProfile, LoginRequest, RegisterRequest } from '@/types/api';

interface AuthContextValue {
  user: UserProfile | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (req: LoginRequest) => Promise<void>;
  register: (req: RegisterRequest) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<UserProfile | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  const login = useCallback(async (req: LoginRequest) => {
    setIsLoading(true);
    try {
      const result = await apiClient.login(req);
      apiClient.setToken(result.accessToken);
      // Store refresh token in sessionStorage (in production: HTTP-only cookie via Next.js route)
      sessionStorage.setItem('refreshToken', result.refreshToken);
      setUser(result.user);
    } finally {
      setIsLoading(false);
    }
  }, []);

  const register = useCallback(async (req: RegisterRequest) => {
    setIsLoading(true);
    try {
      const result = await apiClient.register(req);
      apiClient.setToken(result.accessToken);
      sessionStorage.setItem('refreshToken', result.refreshToken);
      setUser(result.user);
    } finally {
      setIsLoading(false);
    }
  }, []);

  const logout = useCallback(async () => {
    await apiClient.logout().catch(() => {});
    apiClient.clearToken();
    sessionStorage.removeItem('refreshToken');
    setUser(null);
  }, []);

  return (
    <AuthContext.Provider value={{ user, isAuthenticated: !!user, isLoading, login, register, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
```

### web/src/app/login/page.tsx
Clean, centered login form with:
- Email input (type="email", required)
- Password input (type="password", required, show/hide toggle)
- "Login" submit button with loading spinner
- Link to /register
- Error message display (from ApiClientError)
- On success → redirect to `/events`

```typescript
// Key behavior:
// - useAuth().login(req)
// - Catch ApiClientError → display error.message
// - useRouter().push('/events') on success
// - Form validation: both fields required before submit
```

### web/src/app/register/page.tsx
Registration form with:
- Display Name input
- Email input
- Password input (with strength indicator)
- Confirm Password (client-side match validation)
- Submit button with loading state
- Link to /login
- Error display

### web/src/components/Navbar.tsx
Top navigation bar with:
- Logo/brand name
- Navigation links: Events, (authenticated: My Team, My Submission, Judge Queue)
- Auth buttons: "Login" / "Register" when unauthenticated
- User dropdown when authenticated: display name, avatar, "Admin Panel" (if admin), "Logout"

### web/src/components/ProtectedRoute.tsx
```typescript
// Higher-order component / wrapper:
// - If !isAuthenticated, redirect to /login?next={currentPath}
// - If isLoading, show skeleton loader
// - Otherwise render children
```

## Design Requirements
- Use Tailwind CSS for all styling
- Color scheme: dark mode (`bg-gray-900`, `bg-gray-800` cards)
- Forms: centered card, max-w-md, rounded-xl, shadow-2xl
- Primary action button: `bg-indigo-600 hover:bg-indigo-700`
- Input fields: dark background, white text, focus ring indigo
- Error messages: `text-red-400`, icon-prefixed
- Smooth fade-in animation on page load
- Loading spinner (SVG rotating) on submit button
- Mobile responsive (stacks vertically on small screens)

## Tests Required
- TypeScript compilation: `npm run build` → 0 errors
- Visual test: login form renders, submit with empty fields shows validation
- `POST /auth/login` called with correct payload on form submit
- On 401 response → error message displayed, form not cleared
- On success → redirect to /events

## Definition of Done
- [ ] `/login` page renders with correct form
- [ ] `/register` page renders with correct form
- [ ] `useAuth()` hook provides `user`, `isAuthenticated`, `login`, `register`, `logout`
- [ ] `Navbar` shows login/register when unauthenticated, user dropdown when authenticated
- [ ] `ProtectedRoute` redirects to /login for unauthenticated users
- [ ] Error messages from API displayed in UI (not swallowed)
- [ ] `npm run build` → 0 TypeScript errors
- [ ] Dark mode design matching design requirements
