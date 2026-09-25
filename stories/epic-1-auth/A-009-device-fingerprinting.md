---
id: A-009
title: Device Fingerprinting
epic: auth
owner: Unassigned
status: "[ ] backlog"
branch: 
blocks: 
blocked-by: A-001
---

# A-009 · Device Fingerprinting

## Context
- Beyond simple IP-based rate limiting, advanced threat actors rotate IPs.
- Capturing a robust device fingerprint allows us to correlate suspicious activities (e.g., rapid registrations or brute-force logins) across multiple IP addresses but from the same machine.

## What We're Building

A fingerprinting mechanism during authentication and registration to detect anomalous patterns.

1. The frontend generates a robust fingerprint payload (using libraries like FingerprintJS or a custom hashed composite of user-agent, screen resolution, fonts, etc.).
2. The payload is sent with the `POST /api/v1/auth/register` and `/login` requests.
3. The backend calculates a risk score based on historical data for this fingerprint (e.g., number of failed logins, frequency of account creation).
4. If the risk is high, require additional verification (like email OTP) or block the request.

## Definition of Done
- [ ] Add `deviceFingerprint` to Auth API contracts.
- [ ] Implement backend risk scoring logic or integrate with a third-party risk engine.
- [ ] Store device fingerprints alongside refresh tokens or audit logs.
- [ ] Add challenge mechanisms for high-risk fingerprints.
