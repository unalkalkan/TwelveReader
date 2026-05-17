# TwelveReader SaaS Milestones

This document is the working milestone index. The full enriched scope, acceptance criteria, and MVP grouping are in [docs/SAAS_MANIFEST.md](docs/SAAS_MANIFEST.md).

## Development Rule

Each milestone must solve one clear target and avoid cross-target requirements. Earlier milestones create foundations and control points. Later milestones can change without invalidating already-finished work.

## Milestones

1. **SaaS Readiness Baseline**
   - Add `/api/v1` foundation, request IDs, structured errors, health/server-info endpoints, environment modes, and feature flags.
   - No login, quotas, billing, or repository work.

2. **Usage Metering Ledger, Shadow Mode**
   - Record append-only usage events for storage, uploads/imports, segment creation, TTS synthesis, audio bytes, listen minutes, and voices where available.
   - No quota enforcement or billing yet.

3. **Quota Engine, Non-Billing Enforcement**
   - Enforce daily/weekly/monthly quotas from config/manual admin grants.
   - Cover storage, segments, LLM/token use, TTS minutes, new voices, and listening minutes.

4. **Lazy Generation Pipeline**
   - Replace whole-book eager synthesis with `next N segments` generation, resumable jobs, retries, cancellation, partial recovery, and incremental quota use.

5. **Admin Dashboard Shell**
   - Turn Debug Dashboard into `Admin -> Debug` and add Admin sections for overview, jobs, books, storage, billing/usage, support, voices/models, and audit log.

6. **Accounts and Sessions**
   - Add users, roles, sessions, refresh tokens, email magic-link auth, dashboard/admin auth, ownership migration, and audit logs.

7. **Client Server Selection and Login**
   - Add server selection before login, official/custom server validation, token storage/refresh, logout, expired-session UX, and basic usage/quota display.

8. **Private User Library**
   - Add user-scoped books, uploads/imports, audio assets, progress sync, ownership checks, delete/manage flows, and quota-aware private library UX.

9. **Plans, Credits, and Subscriptions Without Stripe**
   - Implement internal plans, subscriptions, credit balances, credit transactions, quota grants, manual admin assignments, and billing auditability.

10. **Stripe Billing Integration**
    - Add Stripe customer creation, checkout, webhooks, subscription sync, invoices, failed-payment handling, and plan upgrade/downgrade flows.

11. **Voice Catalogs**
    - Keep system default voices available and add private user voices with quotas, access control, usage inspection, and admin disable controls.

12. **Exportable Completed Books**
    - Validate export readiness and package completed books in TwelveReader's export format with attached permanent generated audio assets.

13. **Public Repository Format and Official Public-Domain Catalog**
    - Define public repository manifests and make Explore load the official public-domain repo plus user-added public repos.

14. **User Public Repositories**
    - Allow users to publish/unpublish completed books to their own public TwelveReader repo with responsibility modal and admin takedown.

15. **OAuth for Mobile Platforms**
    - Add Google and Apple-compatible mobile login flows, provider verification, account linking, and consistent session behavior.

16. **Private/Authenticated External Repositories**
    - Add basic-auth/static-token/password-protected external repository support, preferably with client-side credential use.

17. **SaaS Operations Hardening**
    - Add backups, restore, monitoring, alerts, migrations, signed URLs, rate limiting, upload validation, legal docs, data export/delete, and deployment rollback processes.

## MVP Scopes

### MVP 0: SaaS Instrumentation MVP

Includes `/api/v1`, health/server-info, request IDs, structured errors, feature flags, usage metering shadow mode, rollups, and debug/admin visibility.

Exit condition: current app works as before, but backend can report resource consumption.

### MVP 1: Quota-Controlled Local SaaS Core

Includes usage metering, quota enforcement, manual grants, quota errors, lazy `next N segments` generation, job visibility, and Admin Dashboard shell.

Exit condition: expensive actions are controlled through quotas and incremental generation, even before real accounts.

### MVP 2: Account-Based Private Library

Includes users, roles, sessions, email magic-link login, admin access control, client server selection/login, user-owned books/jobs/assets/progress, private library, upload/import, playback progress sync, and quota display/errors.

Exit condition: user can select a server, log in, upload/import, generate/listen incrementally, sync progress, and manage a private library under quota limits.

### MVP 3: Internal Billing and Voice Catalog MVP

Includes internal plans, subscriptions, credit balances, credit transactions, quota grants, admin plan/credit operations, default system voices, user private voices, and voice quotas.

Exit condition: admins can simulate paid plans/credits and users can consume resources according to plan/credit rules.

### MVP 4: Paid Hosted SaaS MVP

Includes Stripe customers, checkout for credits/subscriptions, webhooks, invoices, payment failure handling, plan changes, client billing page, and admin billing management.

Exit condition: hosted users can buy credits or subscribe and receive quota grants from successful payments.

### MVP 5: Explore and Public Repository MVP

Includes public repository format, official public-domain repo, Explore repository management, completed book export validation, user public repositories, publish/unpublish, copyright responsibility modal, and admin takedown.

Exit condition: users can browse official books, add public repos, and publish completed books to their own public repo.

### MVP 6: Mobile Auth and Production Hardening MVP

Includes Google/Apple login, account linking, signed URLs, rate limits, upload validation, backup/restore, monitoring/alerts, audit review, legal documents, data export/delete, and production deployment/rollback.

Exit condition: TwelveReader is ready for broader hosted beta usage with mobile-native login and production-grade operations.
