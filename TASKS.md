# TwelveReader SaaS Task Index

The detailed execution backlog now lives on the TwelveReader Kanban board. This file tracks the intended decomposition shape and points to the canonical SaaS manifest.

## Canonical Source

- [docs/SAAS_MANIFEST.md](docs/SAAS_MANIFEST.md)
- [Milestones.md](Milestones.md)
- [Manifest.md](Manifest.md)

## Kanban Decomposition Rule

The board should be structured as:

- MVP parent tasks
  - Milestone child tasks
    - Actual work-task children

All SaaS roadmap tasks should remain **unassigned** until the project owner assigns them. Planning parents and future work may be parked in Triage or Todo depending on the active board workflow.

## MVP Parents

- MVP 0: SaaS Readiness Baseline
- MVP 1: Identity, Sessions, and Ownership Foundation
- MVP 2: Account-Aware Client and Private Library
- MVP 3: Usage Metering and Quota Foundation
- MVP 4: Lazy Generation and Job Management
- MVP 5: Admin Dashboard
- MVP 6: Internal Plans, Credits, and Voice Catalog
- MVP 7: Paid Hosted SaaS
- MVP 8: Explore and Public Repository
- MVP 9: Mobile Auth, Private Repos, and Production Hardening

## Deferred Scope

- Organizations/workspaces
- Team libraries
- Private repository authentication beyond the later external-repo milestone
- Public social discovery
- Recommendations
- Comments/ratings
- Voice marketplace
- App Store / Play Store in-app purchases
- Advanced copyright automation
- Publisher/creator portals
- Federation between TwelveReader servers
