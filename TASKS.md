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

All SaaS roadmap tasks should start in **Triage** and remain **unassigned** until the project owner assigns them and moves them to TODO.

## MVP Parents

- MVP 0: SaaS Instrumentation MVP
- MVP 1: Quota-Controlled Local SaaS Core
- MVP 2: Account-Based Private Library
- MVP 3: Internal Billing and Voice Catalog MVP
- MVP 4: Paid Hosted SaaS MVP
- MVP 5: Explore and Public Repository MVP
- MVP 6: Mobile Auth and Production Hardening MVP

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
