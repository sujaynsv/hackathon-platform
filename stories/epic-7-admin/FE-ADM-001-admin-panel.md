---
id: FE-ADM-001
title: Frontend — Admin Panel
epic: admin
owner: Keerthika
status: "[ ] not-started"
branch: story/FE-ADM-001-admin-panel
blocks: none
blocked-by: ADM-001 (merged to main)
---

# FE-ADM-001 · Frontend — Admin Panel

## Context
- `docs/api-design.md` → Admin section

## What to Build
- Admin-only section behind isAdmin gate
- User list with search/filter
- Audit log browser with action filter
- Impersonation control (input userId, issue token)
- Certificate verification page (public, accessible to anyone with hash URL)

## Definition of Done
- [ ] Admin panel only visible to isAdmin users
- [ ] User list and audit log paginate correctly
- [ ] Certificate verification page works publicly
