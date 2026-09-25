---
id: A-008
title: Turnstile / CAPTCHA
epic: auth
owner: Unassigned
status: "[ ] backlog"
branch: 
blocks: 
blocked-by: A-001
---

# A-008 · Turnstile / CAPTCHA

## Context
- Automated bots can bypass simple rate limits by rotating IPs. 
- Cloudflare Turnstile (or similar invisible CAPTCHA alternatives) provides a privacy-preserving challenge that heavily mitigates bot traffic.

## What We're Building

A requirement for a Turnstile token on the `/register` (and possibly `/login`) endpoints.

1. The frontend invokes Turnstile and includes the `cf-turnstile-response` token in the API request body.
2. The backend verifies this token against Cloudflare's `siteverify` endpoint before processing the registration.

## Definition of Done
- [ ] Update `registerRequest` payload to accept a captcha token.
- [ ] Implement backend HTTP client to verify the token with Cloudflare.
- [ ] Return `400 VALIDATION_ERROR` (or a specific CAPTCHA error) if validation fails.
