---
id: A-005
title: Rate Limiting & Abuse Prevention
epic: auth
owner: Sujay (backend)
status: "[ ] not-started"
branch: story/A-005-rate-limiting
blocks: 
blocked-by: A-001
---

# A-005 · Rate Limiting & Abuse Prevention

## Context
- Registration and login endpoints are highly susceptible to brute force attacks, credential stuffing, and bot spam.
- We need to implement a robust, Redis-backed sliding window rate limiter.
- This fulfills the `Redis 7` cache requirement from the `MASTER-CONTEXT.md`.

## What We're Building

A generic rate-limiting middleware that uses Redis to track request counts per IP address (and potentially per user ID for authenticated routes, though registration relies on IP).

**Configuration:**
- Allow e.g., 5 requests per 15 minutes per IP for `/api/v1/auth/register`.
- Allow e.g., 10 requests per 15 minutes per IP for `/api/v1/auth/login`.

**Response when limited:**
| Condition | HTTP | Code | Message |
|-----------|------|------|---------|
| Rate limit exceeded | 429 | `RATE_LIMITED` | `too many requests` |

## Definition of Done
- [ ] Implement sliding window or fixed window rate limiter using Redis (`go-redis/redis/v8` or `v9`).
- [ ] Create `RateLimitMiddleware` in `internal/shared/middleware/rate_limit.go`.
- [ ] Apply stricter rate limits to the `/auth` group in `cmd/api/main.go`.
- [ ] Write integration test verifying that the 6th request (if limit is 5) receives a 429 status code.
