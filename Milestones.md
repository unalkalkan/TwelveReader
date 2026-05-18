# TwelveReader SaaS Milestones

This document is the working milestone index. The full enriched scope, acceptance criteria, and MVP grouping are in [docs/SAAS_MANIFEST.md](docs/SAAS_MANIFEST.md).

## Development Rule

Each milestone must solve one clear target and avoid cross-target requirements. Earlier milestones create foundations and control points. Later milestones can change without invalidating already-finished work.

## Milestones

0. **SaaS Readiness Baseline**
   - Add `/api/v1` foundation, request IDs, structured errors, health/server-info endpoints, environment modes, and feature flags.
   - Status: completed.
   - No login, quotas, billing, or repository work.

1. **Identity, Sessions, and Ownership Foundation**
   - Add accounts, users, roles, sessions, refresh tokens, email magic-link auth, admin/dashboard auth, ownership migration, ownership helpers, and audit logs.
   - This is the first active SaaS milestone because the rest of the product needs user/account ownership underneath it.

2. **Account-Aware Client and Private Library**
   - Add server selection before login, official/custom server validation, token storage/refresh, logout, expired-session UX, user-scoped books/uploads/audio/progress, private library UX, delete/manage flows, and basic profile/quota display.

3. **Usage Metering and Quota Foundation**
   - Record append-only usage events and rollups, then enforce daily/weekly/monthly quotas from config/manual admin grants.
   - Cover storage, uploads/imports, segments, LLM/token use, TTS, voices, and listening where measurable.

4. **Lazy Generation and Job Management**
   - Replace whole-book eager synthesis with `next N segments` generation, staged jobs, retries, cancellation, partial recovery, per-user job ownership, and incremental quota consumption.

5. **Admin Dashboard**
   - Turn Debug Dashboard into `Admin -> Debug` and add Admin sections for overview, users/accounts, jobs, books, storage, usage/quotas, billing, support, voices/models, audit log, and deployment/status.

6. **Internal Plans, Credits, and Voice Catalog**
   - Implement internal plans, subscriptions, credit balances, credit transactions, quota grants, manual admin assignments, system voice catalog, private user voices, voice quotas, and billing/voice auditability.

7. **Paid Hosted SaaS**
   - Add Stripe customer creation, checkout, webhooks, subscription sync, invoices, failed-payment handling, plan upgrade/downgrade flows, and entitlement/quota grant sync.

8. **Explore and Public Repository**
   - Add completed-book export readiness, TwelveReader export packages, public repository manifests, official public-domain catalog, user public repositories, publish/unpublish, copyright responsibility modal, and admin takedown.

9. **Mobile Auth, Private Repos, and Production Hardening**
   - Add Google/Apple-compatible mobile login flows, provider verification, account linking, private/authenticated external repositories, signed URLs, rate limits, upload validation, backups, restore, monitoring, alerts, legal docs, data export/delete, and deployment rollback processes.

## MVP Scopes

### MVP 0: SaaS Readiness Baseline

Includes `/api/v1`, health/server-info, request IDs, structured errors, feature flags, environment modes, and debug visibility for readiness data.

Exit condition: current app works as before, but the backend has a stable SaaS-ready API/debug foundation.

### MVP 1: Identity, Sessions, and Ownership Foundation

Includes account/user model, roles, sessions, refresh tokens, email magic-link login, admin access control, bootstrap/default user migration, ownership helpers, and audit logs.

Exit condition: users can authenticate, admin pages require admin role, existing local data belongs to a bootstrap/default account, and protected endpoints can enforce user/account ownership.

### MVP 2: Account-Aware Client and Private Library

Includes client server selection, official/custom server validation, login/logout/token refresh, profile/quota display, user-owned books/jobs/assets/progress, private library, upload/import, playback/reading progress sync, and delete/manage flows.

Exit condition: a user can select a server, log in, upload/import a book, generate/listen, sync progress, and manage only their private library.

### MVP 3: Usage Metering and Quota Foundation

Includes append-only usage events, daily/weekly/monthly/all-time rollups, account/user/book/job attribution, quota categories/windows/grants, manual admin grants, quota denial errors, and permissive local/dev mode.

Exit condition: resource usage is visible per account/user and expensive actions can be allowed or denied by quota policy before billing exists.

### MVP 4: Lazy Generation and Job Management

Includes staged import/segment/synthesis/finalize jobs, next-N segment prefetch, retry/cancel/resume/regenerate controls, partial-book usability, user-owned job history, and incremental quota consumption.

Exit condition: uploading/importing a book no longer synthesizes the whole book immediately; users can listen while generation continues and failed work can be retried safely.

### MVP 5: Admin Dashboard

Includes Admin Dashboard navigation, Debug as an admin-only section, overview cards, users/accounts, books, jobs, storage, usage/quota, billing, support, voices/models, audit logs, model health, and deployment/status visibility.

Exit condition: admins can inspect and operate users, books, jobs, quotas, usage, audit events, and system health from the dashboard while retaining all existing debug tools.

### MVP 6: Internal Plans, Credits, and Voice Catalog

Includes internal plans, subscriptions, credit balances, credit transactions, quota grants, admin plan/credit operations, default system voices, user private voices, voice quotas, TTS voice selection, and access control for voice assets.

Exit condition: admins can simulate paid plans/credits and users can consume resources according to internal plan/credit rules while using default or private voices.

### MVP 7: Paid Hosted SaaS

Includes Stripe customers, checkout for credits/subscriptions, idempotent webhooks, subscription sync, invoices, payment failure handling, plan changes, client billing page, admin billing management, and quota grants from successful payments.

Exit condition: hosted users can buy credits or subscribe, and successful payments update internal entitlements without corrupting usage/quota history.

### MVP 8: Explore and Public Repository

Includes completed-book export readiness, TwelveReader export packages, public repository format, official public-domain repository, Explore repository management, user public repositories, publish/unpublish, public/private book state, copyright responsibility modal, and admin takedown.

Exit condition: users can browse official books, add public repos, export completed books, and publish completed books to their own public repo.

### MVP 9: Mobile Auth, Private Repos, and Production Hardening

Includes Google/Apple-compatible login, account linking, private/authenticated external repositories, safe credential storage, signed URLs, rate limiting, upload validation, backup/restore, migrations, monitoring/alerts, legal docs, data export/delete, and production deployment/rollback.

Exit condition: TwelveReader is ready for broader hosted beta usage with mobile-native login, private repository support, protected private assets, and production-grade operational controls.
