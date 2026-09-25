# Technical Specification: Home Page & Email Verification Flow

## 1. Overview & Architecture
This technical specification details the implementation for the Home Page (`/`) and the Email Verification Page (`/verify-email`) in the Next.js frontend, along with the integration of Mailpit for local email testing.

## 2. Component Architecture & Routes

```
web/src/
├── app/
│   ├── page.tsx                          # Home page component (replaces redirect)
│   ├── home.module.css                   # Scoped styles for landing page
│   └── (auth)/
│       └── verify-email/
│           ├── page.tsx                  # Suspense boundary wrapper for query params
│           ├── VerifyEmailClient.tsx     # Client component handling verify logic & states
│           └── verify-email.module.css   # Styles for verification card and states
```

## 3. Detailed Specifications

### 3.1 Home Page (`web/src/app/page.tsx`)
- **Structure**:
  - `<div className={styles.page}>`
    - Hero `<section className={styles.hero}>`:
      - Label badge: `Engineering Platform`
      - Main Headline: `Dogfood Hackathon Platform`
      - Lead text: Clean summary of purpose without marketing fluff.
      - Action cluster: Primary button (`/register` or `/events`), Secondary outline button (`/login`).
    - Feature Grid `<section className={styles.grid}>`:
      - 4 focused blocks with Lucide React icons (`Calendar`, `Users`, `CheckSquare`, `BarChart3`):
        1. **Event Orchestration**: Strict state machine lifecycle from draft to published, active, judging, and closed.
        2. **Team Assembly**: Role-based access control, captaincy delegation, and member invitation flows.
        3. **Calibrated Scoring**: Rubric-driven evaluation with normalized z-score calculations across judges.
        4. **Public Participation & Auditability**: Tamper-proof voting with IP rate-limiting and immutable audit logs.
    - Quick Spec / Architecture Strip:
      - Clean key-value spec table showing tech foundations: `Go 1.23`, `Next.js 14`, `PostgreSQL 16`, `Redis 7`, `MinIO`.
- **Styling (`home.module.css`)**:
  - Uses CSS tokens defined in `globals.css` (`--bg`, `--surface`, `--border`, `--text`, `--text-muted`, `--accent`).
  - No purple gradients or arbitrary box-shadows. Clean 1px solid borders (`var(--border)`).
  - Responsive layout (1 column on mobile, 2 or 4 column grid on desktop).

### 3.2 Email Verification (`web/src/app/(auth)/verify-email/`)
- Next.js 14 App Router requirement: Using `useSearchParams()` requires wrapping inside a `<Suspense>` boundary to prevent de-opting the entire page into client-side rendering.
- **Client Component (`VerifyEmailClient.tsx`)**:
  - **State machine**:
    - `idle`: Token is empty, user enters it manually.
    - `verifying`: HTTP request in-flight.
    - `success`: Account successfully activated (`is_verified = true`).
    - `error`: Failed response with error code and message.
  - **Behavior**:
    - On mount (`useEffect`), if URL parameter `token` is present, immediately execute verification request.
    - If no `token` in URL, present manual input field with label `Verification Token` and submit button.
    - On success: Render success state with button linking to `/login`.
    - On failure: Display user-friendly error (e.g. `INVALID_STATE_TRANSITION` mapped to "Token already used", `DEADLINE_PASSED` mapped to "Verification link expired", or general error) with "Try again" action.
- **API Interaction**:
  - Calls `apiClient.post('/auth/verify-email', { token })`.
  - Re-uses `ApiClientError` from `AuthContext` / `lib/api.ts` for uniform error parsing.

### 3.3 Backend Dev-Mode Logging (`internal/shared/email/stub.go`)
- Update `SendVerificationEmail(ctx context.Context, email, token string)`:
  - Print full clickable verification link:
    ```go
    verifyURL := fmt.Sprintf("http://localhost:3000/verify-email?token=%s", token)
    s.logger.Info("StubSender: verification email generated", "email", email, "verify_url", verifyURL, "token", token)
    ```
  - This allows anyone running the app via Docker (`docker compose logs -f api`) or locally (`go run ./cmd/api`) to directly click the link to verify the account immediately after registration.

## 4. Testing & Verification Plan
1. **Static Analysis & Arch Check**:
   - `npm run lint:arch` (backend architecture check)
   - `npm run lint:api` (Go vet)
   - `npm run test:api` (all unit and integration tests green)
2. **Frontend Type Check & Build**:
   - `cd web && npm run build` (validates TypeScript compilation and SSR/static optimization)
3. **End-to-End Verification**:
   - Visit `http://localhost:3000/` -> displays landing page without redirects or 404s.
   - Register a new account at `http://localhost:3000/register`.
   - Check backend log for `verify_url`.
   - Visit `http://localhost:3000/verify-email?token=<token>` -> verify success state transitions cleanly.
   - Log in at `http://localhost:3000/login` with newly verified user.
