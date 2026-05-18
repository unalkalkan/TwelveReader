# TwelveReader SaaS Milestones and MVP Scopes

## Planning Principle

TwelveReader should become a SaaS in small, isolated steps. Each milestone below has one clear target, produces a usable system state, and avoids depending on unfinished future product areas.

The active sequence now puts identity first. User/account ownership is the foundation for private libraries, quotas, job history, admin operations, billing, and publishing. The completed readiness baseline remains Milestone 0; every active milestone after that should build on authenticated ownership instead of retrofitting it later.

## Product Direction

TwelveReader is an open-source AI audiobook platform.

The hosted TwelveReader service should provide:

- User accounts
- Private user libraries
- Usage metering and quotas
- Credit and subscription-based billing
- TTS generation as a managed backend service
- Admin/support/debug operations
- Public-domain official repositories
- User-published audiobook repositories
- Self-hostable server support
- Client-side server switching
- User voice catalogs

The self-hosted version should remain useful without forcing the official SaaS service. Users should be able to run their own TwelveReader server and point the app at it before login.

---

# Milestone 0: SaaS Readiness Baseline

## Status

Completed.

## Target

Prepare the current system for SaaS work without changing the user-facing product yet.

## Why this comes first

This milestone creates the foundation for safe iteration. It should not introduce accounts, billing, repositories, or new client flows. It only makes the existing system easier to evolve.

## Scope

- Define `/api/v1` namespace for new API work.
- Add request IDs to API responses and logs.
- Add basic structured error format.
- Add environment mode labels: `local`, `dev`, `staging`, `production`.
- Add feature flag mechanism.
- Add server info endpoint for the client.
- Add system health endpoint.
- Keep current ingestion and TTS behavior unchanged.
- Keep current Debug Dashboard unchanged except for linking to new health/server info data.

## API additions

- `GET /api/v1/health`
- `GET /api/v1/server-info`
- `GET /api/v1/features`

## Out of scope

- User accounts
- Login
- Billing
- Quotas
- Admin roles
- Client library changes
- Repository publishing
- Stripe

## Acceptance criteria

- Existing app still works without login.
- Existing Debug Dashboard still works.
- New API endpoints return stable JSON.
- Logs include request IDs.
- Feature flags can be enabled/disabled by config.
- Staging/production configuration can be separated from local/dev configuration.

---

# Milestone 1: Identity, Sessions, and Ownership Foundation

## Target

Introduce real accounts, users, sessions, roles, ownership, and audit trails before building the rest of the SaaS surface.

## Why this comes first after readiness

Private libraries, quotas, billing, admin operations, user voices, jobs, and publishing all need a stable answer to “who owns this?” Building those features before identity would force later migrations and increase security risk.

## Scope

Create account and identity models:

- Account or workspace, even if v1 has one user per account.
- User.
- Role.
- Session.
- Refresh token.
- Audit log entry.

Authentication support:

- Email magic link.
- Session refresh.
- Logout.
- Expired session handling.
- Bootstrap/default admin creation.

Dashboard authentication:

- Admin role.
- User role.
- Role-gated Admin Dashboard routes and APIs.
- No unauthenticated access to admin/debug data in hosted mode.

Ownership foundation:

- Existing local/system data belongs to a default bootstrap account/user.
- New records can be attached to `account_id` and/or `user_id`.
- Ownership helpers/middleware are available for protected APIs.
- Future entities have a clear ownership pattern: books, uploads, audio assets, jobs, progress, voices, usage events, quota grants, billing records, repository publishing state.

Audit foundation:

- Authentication events are auditable.
- Admin-sensitive operations have audit hook points.
- Ownership migration is auditable.

## Out of scope

- Full client private-library migration.
- Google/Apple mobile OAuth.
- Stripe.
- Public repositories.
- Sharing.
- Advanced user profile.
- Multi-user organizations beyond the lightweight account/workspace foundation.

## Acceptance criteria

- Users can authenticate with email magic link.
- Sessions refresh correctly.
- Users and admins have distinct roles.
- Admin Dashboard requires admin access.
- Protected API paths can enforce user/account ownership.
- Existing data remains accessible through a bootstrap/default account and user.
- Audit logs exist for auth/admin-sensitive events.

---

# Milestone 2: Account-Aware Client and Private Library

## Target

Make the TwelveReader client account-aware/server-aware and make the app work as a private, account-based audiobook library.

## Why this follows identity

The backend should own the auth/session/ownership model first. This milestone then applies that model to the client and user library experience.

## Scope

Client server selection:

- Server selection before login.
- Default official TwelveReader server.
- Custom server URL input.
- Server validation using `/api/v1/server-info`.
- Remember selected server.
- Allow changing server after logout.

Client authentication:

- Login screen.
- Email magic-link flow.
- Token storage.
- Authenticated API calls.
- Token refresh.
- Logout.
- Expired session UI.

Client user basics:

- Current user profile screen.
- Basic quota/usage display.
- Error handling for quota denial.

Client library features:

- User-owned book list.
- Upload/import book.
- Book detail page.
- Processing/generation status.
- Delete book.
- Manage generated audio.
- Continue listening.
- Recently added.
- Currently generating.
- Failed/retry state.

Backend library features:

- User-scoped book APIs.
- User-scoped upload/import APIs.
- User-scoped audio asset APIs.
- Ownership checks on every book/audio/progress endpoint.
- No deduplication or shared private objects between users in v1.

Progress sync:

- Reading progress.
- Playback progress.
- Last opened book.
- Last listened segment.

Storage behavior:

- Original file, cover, metadata, segments, and generated audio remain attached to the user-owned book.
- Generated audio is treated as a permanent asset unless the user deletes it.

## Out of scope

- Google/Apple mobile OAuth.
- Stripe checkout.
- Public publishing.
- Explore repository publishing.
- Voice catalog management.
- Organizations/workspaces.
- Sharing between users.

## Acceptance criteria

- User can choose official or custom server before login.
- Client refuses incompatible servers with a clear error.
- User can log in, refresh session, and log out.
- Client API calls use the selected server.
- Each user sees only their own books.
- Users can upload/import, process, listen, and delete their own books.
- Progress sync works across sessions/devices using the same server.
- Quota errors are visible and understandable when quota enforcement is enabled.
- Admin can inspect user books and jobs.

---

# Milestone 3: Usage Metering and Quota Foundation

## Target

Add account-aware usage metering and quota enforcement before billing.

## Why this follows identity and private library

Usage and quota records should be attached to real accounts/users, books, and jobs from the start. That makes billing and admin operations cleaner later.

## Scope

Add append-only usage events for:

- Storage bytes created.
- Book uploads/imports.
- Segment creation.
- LLM/text processing tokens if available.
- TTS synthesis duration in seconds/minutes.
- Generated audio bytes.
- Audio listen minutes if currently measurable.
- Voice creation/import events if available.

Add usage rollups for:

- Daily usage.
- Weekly usage.
- Monthly usage.
- All-time usage.

Usage attribution:

- Account/user.
- Book/job where known.
- Bootstrap/default account for local migrated data.
- System/local user only for local/dev or unauthenticated compatibility paths.

Create quota categories:

- Storage.
- Segment generation.
- LLM/token usage.
- TTS synthesis minutes.
- New voices.
- Audio listen minutes.

Create quota windows:

- Daily.
- Weekly.
- Monthly.

Create quota sources:

- Default free quota.
- Manual admin quota grant.
- Internal/development unlimited quota.

Add quota checks before:

- Upload/import.
- Segment generation.
- TTS synthesis.
- Voice creation/import.
- Audio playback/listening where measurable.

Admin/debug additions:

- View usage events and rollups.
- View quota limits and consumption.
- Manually grant quota.
- Reset quota for test users/system user.
- See blocked actions.
- Recalculate/backfill rollups from raw events.

## Out of scope

- Stripe.
- Invoices.
- Payment failure handling.
- Real subscription plans.
- OAuth.
- Public repositories.
- Client plan screen beyond basic quota display.

## Acceptance criteria

- Every major resource-consuming action emits a usage event.
- Usage events are append-only.
- Usage rollups can be recalculated from raw events.
- Usage can be viewed per account/user and by book/job where known.
- Actions can be allowed or denied by quota policy.
- Quotas work for daily, weekly, and monthly windows.
- Admin can manually grant extra quota.
- Existing local/dev mode can run with unlimited quota.
- Quota denial returns a structured API error.

---

# Milestone 4: Lazy Generation and Job Management

## Target

Replace the current `whole book -> whole segmentation -> whole synthesis` behavior with a fair SaaS-friendly `next N segments` generation and user-owned job model.

## Why this is its own milestone

This is a major backend behavior change. It directly affects quota fairness, infrastructure cost, and user experience. It should be implemented after ownership and quota foundations exist.

## Scope

Change book processing into staged work:

- Import book metadata.
- Prepare enough structure to show the book in the library.
- Segment only what is needed initially.
- Synthesize only the next N required segments.
- Continue generation based on reading/listening position.
- Allow prefetching within a configured limit.

Add job types:

- Book import job.
- Segment preparation job.
- TTS synthesis job.
- Audio stitch/finalize job if needed.

Add job controls:

- Retry failed job.
- Cancel job.
- Resume partial job.
- Mark segment failed.
- Regenerate segment.

Add ownership and quota integration:

- Jobs belong to the requesting account/user.
- Job history is visible to the owning user and admins.
- Segment quota is consumed when segments are generated.
- TTS quota is consumed when audio is synthesized.
- Storage quota is consumed when assets are stored.

## Out of scope

- Stripe.
- Public publishing.
- Repository feeds.
- OAuth.
- Advanced voice catalog.

## Acceptance criteria

- Uploading/importing a book no longer immediately synthesizes the whole book.
- The system can generate only the next required segments.
- Quotas are consumed incrementally.
- A partially generated book remains usable.
- Failed segments can be retried without restarting the whole book.
- User-owned job history is visible to the user and admins.
- Admin/debug view can inspect jobs and segment status.

---

# Milestone 5: Admin Dashboard

## Target

Turn the current Debug Dashboard into the Admin Dashboard while preserving Debug as an admin-only section.

## Why this follows identity, library, usage, and jobs

The dashboard becomes useful once there are real accounts, private books, usage/quota data, and jobs to manage. Building it after those foundations avoids a dashboard shell that needs repeated rewrites.

## Scope

Create Admin Dashboard navigation:

- Overview.
- Users/Accounts.
- Jobs.
- Books.
- Storage.
- Usage/Quotas.
- Billing.
- Support.
- Voices/Models.
- Debug.
- Audit Log.
- Deployment/Status.

Move current Debug Dashboard into:

- `Admin -> Debug`.

Add Overview cards:

- Active jobs.
- Failed jobs.
- Queue depth.
- Storage usage.
- Recent errors.
- Usage consumed today/week/month.
- New users/accounts.
- Recent admin/audit events.

Add operational pages:

- Users/accounts list and detail.
- User book/job/storage inspection.
- Jobs list.
- Book processing status.
- Storage usage summary.
- Usage/quota summary.
- Model/service health.
- Audit log viewer.
- Deployment/server status.

## Out of scope

- Stripe payment collection.
- Impersonation unless explicitly approved later.
- Public repository moderation beyond placeholders.
- Mobile-native OAuth.

## Acceptance criteria

- Existing debug tools are still available under `Admin -> Debug`.
- Admin Dashboard has stable navigation.
- Admin routes and APIs require admin role.
- Dashboard can show users/accounts, usage/quota/job/book data, audit events, and service health.
- No ordinary user can access admin/debug data.

---

# Milestone 6: Internal Plans, Credits, and Voice Catalog

## Target

Implement the internal billing model and user/system voice catalog before integrating real payments.

## Why this comes before Stripe

Stripe should fund existing plan/credit/quota mechanics. It should not define the internal billing model. Voice quotas also depend on the same account/quota foundation.

## Scope

Create internal billing entities:

- Plan.
- Subscription.
- Credit balance.
- Credit transaction.
- Quota grant.
- Billing account state.

Plan behavior:

- Free/default plan.
- Manual paid-like plan assignment.
- Recurring quota grants.
- Credit balance top-up by admin/manual action.
- Credit consumption by usage category if enabled.

Admin dashboard:

- Assign plan.
- Add/remove credits.
- View subscription state.
- View quota grants.
- View usage vs allowance.
- View billing-relevant audit log.

System voices:

- Default voices always available.
- Admin can enable/disable system voices.
- Voices have language and capability metadata.

User voices:

- Add/import user voice.
- List user voices.
- Delete user voice.
- Use user voice for TTS jobs.
- Apply new-voice quota.
- Store voice assets privately.

Voice admin:

- View system voices.
- View user voices.
- Inspect voice usage.
- Disable abusive/problematic voices.

## Out of scope

- Stripe checkout.
- Real payment collection.
- Invoices.
- Payment failure handling.
- App store payments.
- Taxes.
- Public voice marketplace.
- Sharing voices between users.
- Commercial voice licensing automation.
- Advanced voice moderation.

## Acceptance criteria

- Plans add quota to users/accounts.
- Credits can be granted and consumed internally.
- Daily/weekly/monthly quotas still protect against abuse.
- Admin can simulate paid users without Stripe.
- Billing state is auditable.
- Users can use default voices.
- Users can add private voices if quota allows.
- TTS jobs can select default or user-owned voices.
- Voice assets are access-controlled.
- Admin can inspect and disable voices.

---

# Milestone 7: Paid Hosted SaaS

## Target

Connect the internal billing model to real payments.

## Scope

Stripe features:

- Customer creation.
- Checkout for credit purchase.
- Checkout for subscription purchase.
- Webhook handling.
- Subscription status sync.
- Payment success handling.
- Payment failure handling.
- Plan upgrade/downgrade.
- Invoice links/history.

Internal integration:

- Stripe events create credit transactions or subscription changes.
- Stripe subscriptions create quota grants.
- Stripe payment failures affect billing state but do not corrupt usage history.
- Admin can view Stripe-linked billing state.

Client/Admin surfaces:

- Client billing page.
- Checkout entry points.
- Invoice/payment state visibility.
- Admin billing management views.

## Out of scope

- App Store / Play Store in-app purchases.
- Organizations/workspaces.
- Repository publishing.
- Advanced tax handling beyond Stripe defaults.

## Acceptance criteria

- User can buy credits or a subscription through Stripe.
- Stripe webhooks update internal billing state idempotently.
- Failed payments are visible to user/admin.
- Quotas are granted from successful billing events.
- Manual admin overrides still work.

---

# Milestone 8: Explore and Public Repository

## Target

Deliver the completed-book export flow and open audiobook repository model.

## Scope

Completion and export rules:

- Book metadata complete.
- Segments complete.
- Synths/audio complete.
- Assets attached to the book.
- Export manifest valid.
- Export package can be rebuilt if assets changed.

Export features:

- Export private book package.
- Validate export package.
- Show export readiness in client and admin.
- Download/export action.
- Clear error state if book is not exportable.

Repository format:

- Repository manifest.
- Book list.
- Book metadata.
- Asset URLs.
- Cover/audio references.
- Version/compatibility fields.

Official TwelveReader repository:

- Embedded as default Explore repository.
- Contains only public-domain books.
- Can be removed by the user.
- Can be re-added later.

Client Explore features:

- Repository list.
- Add repository URL.
- Remove repository.
- Refresh repository.
- Browse repository books.
- Import/listen from repository depending on existing client model.

Publishing rules:

- Books are private by default.
- Only completed/exportable books can be made public.
- User must accept copyright/responsibility modal before publishing.
- Public book can be unpublished.
- Public repository only exposes books marked public.

User repo identity:

- Each user has a TwelveReader Repo.
- Repo address uses a format that can identify system and user/repo.
- Official hosted server repo identity is distinct from self-hosted repo identity.

Possible repo naming pattern:

- `official.twelvereader://username/main`
- `server-domain.example/@username/main`
- Final scheme can be changed later, but it must separate server/system identity from user/repo identity.

Backend features:

- Public repo endpoint per user.
- Public book manifest generation.
- Public asset access for published books.
- Unpublish flow.
- Takedown/remove flow for admin.

Client features:

- Publish/unpublish button.
- Public/private book state.
- Copy repo URL.
- Add another user's repo to Explore.

## Out of scope

- Private/password-protected repositories.
- Organization repositories.
- Social discovery.
- Recommendations.
- Comments/ratings.
- Private repository auth.
- Moderation workflows beyond admin takedown.

## Acceptance criteria

- A completed book can be exported in the TwelveReader format.
- Incomplete books cannot be marked export-ready.
- Export validation errors are visible.
- Export package contains the permanent generated audio assets attached to the book.
- Explore can load the official public-domain repository.
- User can add/remove custom public repository URLs.
- Repository compatibility is validated.
- Broken repositories fail gracefully.
- Official repository is default but not mandatory.
- User can publish a completed book.
- User cannot publish incomplete books.
- Published books appear in that user's repo feed.
- Other users can add that repo URL in Explore.
- Unpublished books disappear from public repo output.
- Admin can remove public content if required.

---

# Milestone 9: Mobile Auth, Private Repos, and Production Hardening

## Target

Make the hosted TwelveReader service reliable enough for broader production use and mobile distribution.

## Scope

Authentication providers:

- Google / Google account login.
- Apple-compatible login path where technically appropriate.
- Account linking to existing email magic-link accounts.
- Provider token verification.
- Provider-specific session creation.

Client auth features:

- Continue with Google.
- Continue with Apple where supported.
- Link/unlink provider if safe.
- Login error handling.

Private/authenticated external repositories:

- Basic HTTP auth.
- Static bearer token/header.
- Password-protected repository config stored locally in the client.
- Add private repo URL.
- Enter credentials.
- Validate repository.
- Refresh private repository.
- Remove credentials when repo is removed.
- Do not send private repo credentials to official TwelveReader servers unless explicitly required by design.
- Prefer client-side repository fetching for external private repos if possible.

Operational hardening:

- Backups.
- Restore process.
- Database migrations.
- Object storage lifecycle rules.
- Orphaned asset cleanup.
- Monitoring.
- Alerts.
- Error-rate dashboards.
- Queue dashboards.
- Deployment status.
- Rollback procedure.
- Admin audit log review.

Security hardening:

- Rate limiting.
- Abuse detection.
- Upload validation.
- Malware scanning if feasible.
- Signed URLs for private assets.
- Secrets management.
- Sensitive log redaction.
- Data export/delete support.

Legal/compliance basics:

- Terms of Service.
- Privacy Policy.
- Copyright policy.
- DMCA/contact process.
- User responsibility modal for uploads/publishing.
- Admin takedown flow.

## Out of scope

- New product features unrelated to hardening/auth/private repos.
- Organizations/workspaces.
- Social discovery.
- Marketplace features.
- App Store / Play Store in-app purchases.
- Full OAuth for external repositories.
- Repository federation.
- Shared team repositories.

## Acceptance criteria

- Users can log in with supported native providers.
- Existing email users can link provider login.
- Sessions behave the same regardless of provider.
- Admin can see auth provider metadata without exposing sensitive tokens.
- User can add a basic-auth protected repo.
- Credentials are stored safely according to client platform constraints.
- Invalid credentials fail clearly.
- Public repo behavior remains unchanged.
- Production backup and restore process is tested.
- Private assets are not publicly accessible without signed URLs or authorization.
- Admin actions are audited.
- Abuse/rate limits are active.
- Legal documents and upload/publish responsibility modals exist.
- Alerts notify maintainers of critical failures.

---

# MVP Scopes

## MVP 0: SaaS Readiness Baseline

## Purpose

Keep the already-completed SaaS readiness baseline as the stable foundation.

## Includes

- `/api/v1` foundation.
- Health/server-info endpoints.
- Request IDs.
- Structured errors.
- Feature flags.
- Environment modes.
- Debug visibility for readiness endpoints.

## Excludes

- Login.
- Billing.
- Enforced quotas.
- Client account flow.

## Exit condition

The current app works as before, and the backend has a stable SaaS-ready API/debug foundation.

---

## MVP 1: Identity, Sessions, and Ownership Foundation

## Purpose

Establish the account, session, role, ownership, and audit model that the rest of the SaaS depends on.

## Includes

- Accounts/workspaces.
- Users.
- Roles: user/admin.
- Sessions and refresh tokens.
- Email magic-link login.
- Admin Dashboard access control.
- Bootstrap/default account and user.
- Ownership helpers and middleware.
- Ownership migration for existing local/system data.
- Audit log model and auth/admin event hooks.

## Excludes

- Stripe.
- Public publishing.
- Full client private-library migration.
- Mobile-native OAuth.
- User voice catalog.

## Exit condition

Users can authenticate, admin pages require admin role, protected APIs can enforce account/user ownership, and existing local data belongs to a bootstrap/default account.

---

## MVP 2: Account-Aware Client and Private Library

## Purpose

Turn TwelveReader into a private, account-based hosted service from the user's point of view.

## Includes

- Client server selection.
- Official/custom server validation.
- Client login/logout.
- Token storage and refresh.
- Expired-session handling.
- User profile and basic quota display.
- User-owned books.
- User-owned jobs/assets/progress.
- Private library.
- Upload/import.
- Book detail/status.
- Playback/reading progress sync.
- Delete/manage books.

## Excludes

- Stripe.
- Public publishing.
- Explore repository publishing.
- Mobile-native OAuth.
- User voice catalog.

## Exit condition

A user can select a server, log in, upload/import a book, generate/listen, sync progress, and manage only their own private library.

---

## MVP 3: Usage Metering and Quota Foundation

## Purpose

Make resource usage measurable and enforceable per account/user before plans or payments exist.

## Includes

- Append-only usage events.
- Usage attribution to account/user/book/job.
- Daily/weekly/monthly/all-time rollups.
- Recalculation/backfill.
- Quota categories and windows.
- Manual quota grants.
- Local/dev unlimited mode.
- Quota denial errors.
- Admin/debug views for usage, quota, and blocked actions.

## Excludes

- Stripe.
- Real subscription plans.
- OAuth.
- Public repositories.

## Exit condition

Resource usage is visible per account/user, and expensive actions can be controlled through quota policy without billing.

---

## MVP 4: Lazy Generation and Job Management

## Purpose

Make generation scalable and fair by replacing whole-book eager synthesis with staged, resumable, user-owned jobs.

## Includes

- Staged import/segment/synthesis/finalize jobs.
- Next-N segment generation and prefetch.
- Retry/cancel/resume/regenerate controls.
- Partial-book usability.
- User-owned job history.
- Incremental quota consumption.
- Admin/user job visibility.

## Excludes

- Stripe.
- Public repositories.
- OAuth.
- Advanced voice catalog.

## Exit condition

Book upload/import does not synthesize the whole book immediately; users can listen while generation continues; failed segments/jobs can be retried safely.

---

## MVP 5: Admin Dashboard

## Purpose

Convert the Debug Dashboard into a real Admin Dashboard for operating the hosted SaaS.

## Includes

- Admin Dashboard shell and navigation.
- Debug as `Admin -> Debug`.
- Overview cards.
- Users/accounts views.
- Jobs/books/storage views.
- Usage/quota views.
- Billing/support placeholders or internal views.
- Voices/models health views.
- Audit log viewer.
- Deployment/server status.

## Excludes

- Stripe payment collection.
- Mobile-native OAuth.
- Impersonation unless explicitly approved.

## Exit condition

Admins can inspect and operate users, books, jobs, quotas, usage, audit events, and system health while existing debug tools remain available under admin access control.

---

## MVP 6: Internal Plans, Credits, and Voice Catalog

## Purpose

Add the SaaS business model internally and support system/private voices before collecting real payments.

## Includes

- Plans.
- Subscriptions as internal records.
- Credit balances.
- Credit transactions.
- Quota grants from plans/credits.
- Admin plan assignment.
- Admin credit grants.
- Default system voices.
- User private voices.
- Voice quotas.
- TTS voice selection.
- Voice access control and admin inspection.

## Excludes

- Stripe.
- App store payments.
- Public voice marketplace.
- Public repositories.

## Exit condition

Admins can simulate paid plans and credit-based usage. Users can consume resources according to internal plan/credit rules and use default or private voices.

---

## MVP 7: Paid Hosted SaaS

## Purpose

Connect the internal billing system to real money.

## Includes

- Stripe customer creation.
- Credit purchase checkout.
- Subscription checkout.
- Stripe webhooks.
- Invoices.
- Payment failure handling.
- Plan upgrade/downgrade.
- Billing page in client.
- Billing management in Admin Dashboard.
- Quota grants from successful payments.

## Excludes

- App Store / Play Store purchases.
- Organizations/workspaces.
- Repository publishing.

## Exit condition

A hosted user can buy credits or subscribe, and the system automatically grants quotas based on successful payments.

---

## MVP 8: Explore and Public Repository

## Purpose

Deliver completed-book export and the open audiobook repository model.

## Includes

- Completed book export validation.
- TwelveReader export package generation.
- Public repository format.
- Official TwelveReader public-domain repository.
- Client Explore repository list.
- Add/remove custom public repositories.
- User public repositories.
- Publish/unpublish completed books.
- Public/private book state.
- Copyright responsibility modal.
- Admin takedown flow.

## Excludes

- Private/password-protected repositories.
- Organization repositories.
- Social discovery.
- Recommendations.

## Exit condition

Users can export completed books, browse the official public-domain repository, add other public repositories, and publish completed books to their own public repository.

---

## MVP 9: Mobile Auth, Private Repos, and Production Hardening

## Purpose

Make the product ready for wider mobile distribution and production operations.

## Includes

- Google login.
- Apple login where applicable.
- Account linking.
- Private/authenticated external repositories.
- Safe credential storage.
- Signed URLs for private assets.
- Rate limiting.
- Upload validation.
- Backup/restore.
- Monitoring/alerts.
- Audit log review tools.
- Terms/Privacy/Copyright/DMCA documents.
- Data export/delete.
- Production deployment and rollback process.

## Excludes

- Organizations/workspaces.
- Marketplace features.
- App Store / Play Store in-app purchases.

## Exit condition

TwelveReader is ready for broader hosted beta usage with mobile-native login, private repository support, protected private assets, and production-grade operational controls.

---

# Deferred / Later Scope

These are intentionally not part of the early SaaS milestones:

- Organizations/workspaces beyond the lightweight account/workspace foundation.
- Team libraries.
- Public social discovery.
- Recommendations.
- Comments/ratings.
- Voice marketplace.
- App Store / Play Store in-app purchases.
- Advanced copyright automation.
- Publisher/creator portals.
- Federation between TwelveReader servers.

---

# Recommended Development Order

1. Milestone 0: SaaS Readiness Baseline.
2. Milestone 1: Identity, Sessions, and Ownership Foundation.
3. Milestone 2: Account-Aware Client and Private Library.
4. Milestone 3: Usage Metering and Quota Foundation.
5. Milestone 4: Lazy Generation and Job Management.
6. Milestone 5: Admin Dashboard.
7. Milestone 6: Internal Plans, Credits, and Voice Catalog.
8. Milestone 7: Paid Hosted SaaS.
9. Milestone 8: Explore and Public Repository.
10. Milestone 9: Mobile Auth, Private Repos, and Production Hardening.

This order keeps each milestone focused while putting account ownership at the start of active SaaS work. Later milestones can change without invalidating the completed readiness baseline or the identity foundation.
