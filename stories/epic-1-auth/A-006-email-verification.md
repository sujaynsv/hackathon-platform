---
id: A-006
title: Email Verification Loop
epic: auth
owner: Unassigned
status: "[x] done"
branch: 
blocks: 
blocked-by: A-001
---

# A-006 · Email Verification Loop

## Context
- To ensure users actually own their email addresses, we should require verification before they are fully authorized to perform critical actions (like joining teams or submitting projects).
- Currently, `NewUser` defaults `is_active=true`. We should potentially add `is_verified=false` and only toggle it upon verification.

## What We're Building

An email verification flow involving magic links or OTPs.

1. When a user registers, generate a secure token (stored in Redis or DB) and email it to them.
2. Provide a new endpoint: `POST /api/v1/auth/verify-email`.
3. If the token is valid, mark `is_verified = true`.
4. (Optional) Enforce `is_verified` via a new middleware for specific protected routes.

## Definition of Done
- [x] Update User domain model with `IsVerified` flag.
- [x] Create `POST /api/v1/auth/verify-email` endpoint.
- [x] Add email sending capabilities (e.g., SMTP or AWS SES stub).
- [x] Add OTP generation and verification logic.
