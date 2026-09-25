# Product Requirements Document (PRD): Home Page & Email Verification Flow

## 1. Executive Summary & Problem Statement
Currently, navigating to the root URL (`/`) triggers a redirect to `/events`, which returns a 404 because the Events module (FE-E-001) has not been implemented yet. Users arriving at the application are met with an error page instead of a purposeful landing experience.
Furthermore, while user registration (FE-A-001) returns `requiresVerification: true` and the backend has an email verification endpoint (`POST /api/v1/auth/verify-email`), the platform lacks:
1. A dedicated frontend verification page (`/verify-email`) to process magic link tokens and manual token submissions.
2. Development-mode visibility for verification tokens via Mailpit, allowing developers to test account activation without direct database inspection.

## 2. Goals & Objectives
- **Professional Home Page (`/`)**: Provide a typography-first, high-contrast, structured landing page communicating platform capabilities (Event Lifecycle, Teams, Submissions, Rubric Judging, and Public Voting) with clear calls to action (Sign In, Create Account, Browse Events).
- **Email Verification Page (`/verify-email`)**: A dedicated page supporting automatic verification via URL parameter (`?token=...`) and manual token entry, providing clear pending, success, and error feedback.
- **Developer Experience for Email Verification**: Ensure developers running locally or via Docker can immediately see the verification link/token in the backend console logs so testing is seamless without third-party email infrastructure.
- **Strict Compliance with Anti-Sloth Design Standards**:
  - No neon color palettes or arbitrary glows
  - No emojis anywhere in copy, icons, or navigation (Lucide React SVG icons only)
  - No decorative purple gradients
  - Purposeful spacing (8px grid) and contrast-compliant typography

## 3. User Personas & User Journeys

### Persona 1: New Participant / Developer
1. Arrives at `http://localhost:3000/`. Sees the platform overview and clicks **"Get Started"** or **"Create Account"**.
2. Fills out registration form at `/register`.
3. Sees notification: "Account created. Check your email (or dev server console) for your verification link."
4. Clicks the verification link or navigates to `/verify-email?token=<token>`.
5. The page automatically validates the token against the API, transitions to a success screen, and offers a 1-click **"Sign in to your account"** button.

### Persona 2: Existing User
1. Arrives at `http://localhost:3000/`.
2. Sees platform navigation and clicks **"Sign In"** in Navbar or Hero.
3. Authenticates and is taken to the platform.

## 4. Scope & Functional Requirements

### 4.1 Home Page (`/`)
- **Hero Section**:
  - Wordmark and title: "Dogfood Hackathon Platform"
  - Subheading describing purpose: A developer-first, offline-capable platform for hosting and scoring hackathons.
  - Primary Action: "Create Account" (`/register`) or "Browse Events" (`/events`).
  - Secondary Action: "Sign In" (`/login`).
- **Core Pillars (Structured Grid / Section)**:
  - Event Management & Lifecycle
  - Team Formation & Rosters
  - Rubric Scoring & Standardized Evaluation
  - Public Voting & Audit Logging
- **Footer**: Clean technical footer with version info, API status, and documentation links.

### 4.2 Email Verification (`/verify-email`)
- **Route**: `/verify-email`
- **Inputs**: Reads `token` query param (`useSearchParams`) or allows pasting token into a text field.
- **Actions**: Calls `POST /api/v1/auth/verify-email` with `{ "token": "<token>" }`.
- **States**:
  - *Idle / Manual Entry*: Form with Token input and "Verify Email" button.
  - *Verifying (Loading)*: Accessible spinner and status text.
  - *Success*: Confirmation banner and "Proceed to Sign In" button (`/login`).
  - *Error*: Explanatory error message (e.g. "Token expired", "Token already used", or "Invalid token") with an option to re-enter a token.

### 4.3 Backend Dev-Mode Logging for Email Verification
- In `internal/shared/email/stub.go`, when `StubSender.SendVerificationEmail` is invoked, log the full verification URL:
  `http://localhost:3000/verify-email?token=<rawToken>` so the developer can click it directly from terminal logs or Docker logs (`docker compose logs -f api`).

## 5. Non-Functional Requirements & Invariants
- Zero emojis across all UI copy and code comments.
- Adhere strictly to [docs/frontend-standards.md](file:///Users/sujaynimmagadda/Documents/Projects/Hackathon-DogFood/docs/frontend-standards.md) design tokens and CSS module encapsulation.
- Keyboard accessible (WCAG AA), semantic HTML tags (`<main>`, `<header>`, `<section>`, `<form>`).
