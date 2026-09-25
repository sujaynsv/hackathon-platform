---
id: A-004
title: Compromised Password Check (HaveIBeenPwned)
epic: auth
owner: Sujay (backend)
status: "[ ] not-started"
branch: story/A-004-pwned-passwords
blocks: 
blocked-by: A-001
---

# A-004 · Compromised Password Check (HaveIBeenPwned)

## Context
- We want to prevent users from registering with passwords that are already known to be breached.
- The *HaveIBeenPwned* k-anonymity API allows us to securely check if a password is breached without sending the actual password to a third party (we only send the first 5 characters of the SHA-1 hash).
- If the password appears in the database, we reject the registration with `400 VALIDATION_ERROR`.

## What We're Building

A new step in the `RegisterService` (and optionally `LoginService` or `PasswordResetService` later) that performs a GET request to `https://api.pwnedpasswords.com/range/{prefix}`.

**Error case addition to POST /api/v1/auth/register:**
| Condition | HTTP | Code | Message |
|-----------|------|------|---------|
| Password breached | 400 | `VALIDATION_ERROR` | `password has appeared in a data breach` |

## Definition of Done
- [ ] Create an outbound port `PasswordValidator` interface in `internal/auth/port/out.go`.
- [ ] Implement `HIBPValidator` in `internal/shared/auth.go` or a similar shared location.
- [ ] Integrate into `RegisterService` to block registration if the password is breached.
- [ ] Add unit test for the validator (mocking the HTTP call if necessary).
- [ ] Update `A-001` Postman collection to include a breached password failure case.
