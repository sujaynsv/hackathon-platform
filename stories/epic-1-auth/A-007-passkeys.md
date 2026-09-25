---
id: A-007
title: Passkeys (WebAuthn)
epic: auth
owner: Unassigned
status: "[x] done"
branch: 
blocks: 
blocked-by: A-001
---

# A-007 · Passkeys (WebAuthn)

## Context
- Modern passwordless authentication significantly reduces the risk of phishing and credential stuffing.
- We want to allow users to register and log in using their device's native biometric sensors (FaceID, TouchID) via the WebAuthn standard.

## What We're Building

A WebAuthn relying party implementation on the backend, alongside frontend hooks to trigger the passkey API.

1. `POST /api/v1/auth/webauthn/register/begin` & `/finish`
2. `POST /api/v1/auth/webauthn/login/begin` & `/finish`
3. Store WebAuthn credentials associated with the user account in Postgres.

## Definition of Done
- [ ] Create WebAuthn credential tables in Postgres.
- [ ] Implement begin/finish flows for both registration and authentication.
- [ ] Fallback to traditional passwords if WebAuthn is unavailable or declined.
