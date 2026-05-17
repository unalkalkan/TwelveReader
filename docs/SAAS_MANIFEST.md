# TwelveReader SaaS Milestones and MVP Scopes

## Planning Principle

TwelveReader should become a SaaS in small, isolated steps. Each milestone below has one clear target, produces a usable system state, and avoids depending on unfinished future product areas.

The sequence starts from the smallest practical change to the current system and grows toward the full SaaS model: hosted accounts, quotas, admin operations, client login, repositories, billing, and public publishing.

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

# Milestone 1: Usage Metering Ledger, Shadow Mode

## Target

Add a usage metering system that records what the current system already does, without blocking user actions yet.

## Why this comes before billing

Billing should not be implemented until usage is measurable. This milestone creates the accounting layer first, but runs it in shadow mode so it cannot break the product.

## Scope

Add append-only usage events for:

- Storage bytes created
- Book uploads/imports
- Segment creation
- LLM/text processing tokens if available
- TTS synthesis duration in seconds/minutes
- Generated audio bytes
- Audio listen minutes if currently measurable
- Voice creation/import events if available

Add usage rollups for:

- Daily usage
- Weekly usage
- Monthly usage
- All-time usage

Add admin/debug visibility for:

- Recent usage events
- Usage totals by anonymous/system user
- Usage totals by book/job where known
- Failed or incomplete metering events

## Important design choice

Until real accounts exist, usage can be attributed to a temporary `system_user` or `local_user`. This keeps the milestone isolated from authentication.

## Out of scope

- Enforcing quotas
- Charging users
- Stripe
- User login
- Plan management
- Client changes

## Acceptance criteria

- Every major resource-consuming action emits a usage event.
- Usage events are append-only.
- Usage rollups can be recalculated from raw events.
- Existing app behavior is unchanged.
- Admin/debug view can show usage for recent operations.

---

# Milestone 2: Quota Engine, Non-Billing Enforcement

## Target

Add quota rules and enforcement without adding payments yet.

## Why this is isolated

The system learns to decide whether an action is allowed based on quota. The source of quota is still simple config or manual admin grants, not Stripe.

## Scope

Create quota categories:

- Storage
- Segment generation
- LLM/token usage
- TTS synthesis minutes
- New voices
- Audio listen minutes

Create quota windows:

- Daily
- Weekly
- Monthly

Create quota sources:

- Default free quota
- Manual admin quota grant
- Internal/development unlimited quota

Add quota checks before:

- Upload/import
- Segment generation
- TTS synthesis
- Voice creation/import
- Audio playback/listening where measurable

Add user-facing structured quota errors, even if the current client only shows generic error text initially.

## Admin/debug additions

- View quota limits
- View quota consumption
- Manually grant quota
- Reset quota for test users/system user
- See blocked actions

## Out of scope

- Stripe
- Invoices
- Payment failure handling
- Real subscription plans
- OAuth
- Public repositories
- Client plan screen

## Acceptance criteria

- Actions can be allowed or denied by quota policy.
- Quotas work for daily, weekly, and monthly windows.
- Admin can manually grant extra quota.
- Existing local/dev mode can run with unlimited quota.
- Quota denial returns a structured API error.

---

# Milestone 3: Lazy Generation Pipeline

## Target

Replace the current `whole book -> whole segmentation -> whole synthesis` behavior with a fair SaaS-friendly `next N segments` generation model.

## Why this is its own milestone

This is a major backend behavior change. It directly affects quota fairness and infrastructure cost. It should be completed before large-scale user onboarding or paid billing.

## Scope

Change book processing into staged work:

- Import book metadata.
- Prepare enough structure to show the book in the library.
- Segment only what is needed initially.
- Synthesize only the next N required segments.
- Continue generation based on reading/listening position.
- Allow prefetching within a configured limit.

Add job types:

- Book import job
- Segment preparation job
- TTS synthesis job
- Audio stitch/finalize job if needed

Add job controls:

- Retry failed job
- Cancel job
- Resume partial job
- Mark segment failed
- Regenerate segment

Add quota integration:

- Segment quota is consumed when segments are generated.
- TTS quota is consumed when audio is synthesized.
- Storage quota is consumed when assets are stored.

## Out of scope

- User login
- Stripe
- Public publishing
- Repository feeds
- OAuth
- Advanced voice catalog

## Acceptance criteria

- Uploading/importing a book no longer immediately synthesizes the whole book.
- The system can generate only the next required segments.
- Quotas are consumed incrementally.
- A partially generated book remains usable.
- Failed segments can be retried without restarting the whole book.
- Admin/debug view can inspect jobs and segment status.

---

# Milestone 4: Admin Dashboard Shell

## Target

Turn the current Debug Dashboard into the beginning of the Admin Dashboard while preserving Debug as a section.

## Why this milestone is isolated

It changes dashboard structure and operations visibility, but does not require real users or billing yet.

## Scope

Create Admin Dashboard navigation:

- Overview
- Jobs
- Books
- Storage
- Billing/Usage
- Support
- Voices/Models
- Debug
- Audit Log

Move current Debug Dashboard into:

- `Admin -> Debug`

Add Overview cards:

- Active jobs
- Failed jobs
- Queue depth
- Storage usage
- Recent errors
- Usage consumed today/week/month

Add operational pages:

- Jobs list
- Book processing status
- Storage usage summary
- Usage/quota summary
- Model/service health

## Out of scope

- Admin login
- Role permissions
- Real user management
- Stripe
- Impersonation
- Client changes

## Acceptance criteria

- Existing debug tools are still available.
- Admin Dashboard has stable navigation.
- Dashboard can show usage/quota/job data from earlier milestones.
- No user-facing client behavior changes.

---

# Milestone 5: Accounts and Sessions

## Target

Introduce real users, sessions, and ownership without changing the whole client experience at once.

## Scope

Create account model:

- User
- Role
- Session
- Refresh token
- Audit log entry

Authentication support:

- Email magic link
- Session refresh
- Logout
- Expired session handling

Dashboard authentication:

- Google OAuth for dashboard/admin if preferred
- Admin role
- User role
- Role-gated Admin Dashboard pages

Ownership migration:

- Existing local/system data belongs to a default bootstrap user.
- New books belong to the authenticated user.
- Usage and quota events are attributed to users.

## Out of scope

- Google Play login
- Apple Game Center login
- Stripe
- Public repositories
- Sharing
- Advanced user profile
- Multi-user organizations/workspaces

## Acceptance criteria

- Users can authenticate with email magic link.
- Sessions refresh correctly.
- Users own their books, jobs, usage, and quota records.
- Admin Dashboard requires admin access.
- User and admin roles are enforced.
- Existing data remains accessible through a bootstrap/default user.

---

# Milestone 6: Client Server Selection and Login

## Target

Make the TwelveReader client account-aware and server-aware.

## Why this is separate from account backend

The backend can support accounts before the client is fully migrated. This milestone focuses only on the client experience.

## Scope

Client server selection:

- Server selection before login
- Default official TwelveReader server
- Custom server URL input
- Server validation using `/api/v1/server-info`
- Remember selected server
- Allow changing server after logout

Client authentication:

- Login screen
- Email magic link flow
- Token storage
- Authenticated API calls
- Token refresh
- Logout
- Expired session UI

Client user basics:

- Current user profile screen
- Basic quota/usage display
- Error handling for quota denial

## Out of scope

- Google Play OAuth
- Apple Game Center OAuth
- Stripe checkout
- Public repository browser changes
- Publishing books
- Voice catalog management

## Acceptance criteria

- User can choose official or custom server before login.
- Client refuses incompatible servers with a clear error.
- User can log in, refresh session, and log out.
- Client API calls use the selected server.
- Quota errors are visible and understandable.

---

# Milestone 7: Private User Library

## Target

Make the app work as a private, account-based audiobook library.

## Scope

Client library features:

- User-owned book list
- Upload/import book
- Book detail page
- Processing/generation status
- Delete book
- Manage generated audio
- Continue listening
- Recently added
- Currently generating
- Failed/retry state

Backend library features:

- User-scoped book APIs
- User-scoped upload/import APIs
- User-scoped audio asset APIs
- Ownership checks on every book/audio/progress endpoint
- No deduplication
- No shared objects between users

Progress sync:

- Reading progress
- Playback progress
- Last opened book
- Last listened segment

Storage behavior:

- Original file, cover, metadata, segments, and generated audio remain attached to the user-owned book.
- Generated audio is treated as a permanent asset unless the user deletes it.

## Out of scope

- Public publishing
- Explore repositories
- Stripe
- OAuth providers beyond magic link
- Organizations/workspaces
- Sharing between users

## Acceptance criteria

- Each user sees only their own books.
- Users can upload/import, process, listen, and delete their own books.
- Progress sync works across sessions/devices using the same server.
- Quotas are applied to upload, segmentation, TTS, storage, and listening.
- Admin can inspect user books and jobs.

---

# Milestone 8: Plans, Credits, and Subscriptions Without Stripe

## Target

Implement the internal billing model before integrating a payment provider.

## Why this comes before Stripe

Stripe should only fund existing plan/credit/quota mechanics. It should not define the internal billing model.

## Scope

Create internal billing entities:

- Plan
- Subscription
- Credit balance
- Credit transaction
- Quota grant
- Billing account state

Plan behavior:

- Free/default plan
- Manual paid-like plan assignment
- Recurring quota grants
- Credit balance top-up by admin/manual action
- Credit consumption by usage category if enabled

Admin dashboard:

- Assign plan
- Add/remove credits
- View subscription state
- View quota grants
- View usage vs allowance
- View billing-relevant audit log

## Out of scope

- Stripe checkout
- Real payment collection
- Invoices
- Payment failure handling
- App store payments
- Taxes

## Acceptance criteria

- Plans add quota to users.
- Credits can be granted and consumed internally.
- Daily/weekly/monthly quotas still protect against abuse.
- Admin can simulate paid users without Stripe.
- Billing state is auditable.

---

# Milestone 9: Stripe Billing Integration

## Target

Connect the internal billing model to real payments.

## Scope

Stripe features:

- Customer creation
- Checkout for credit purchase
- Checkout for subscription purchase
- Webhook handling
- Subscription status sync
- Payment success handling
- Payment failure handling
- Plan upgrade/downgrade
- Invoice links/history

Internal integration:

- Stripe events create credit transactions or subscription changes.
- Stripe subscriptions create quota grants.
- Stripe payment failures affect billing state but do not corrupt usage history.
- Admin can view Stripe-linked billing state.

## Out of scope

- App Store / Play Store in-app purchases
- Organizations/workspaces
- Repository publishing
- Advanced tax handling beyond Stripe defaults

## Acceptance criteria

- User can buy credits or a subscription through Stripe.
- Stripe webhooks update internal billing state idempotently.
- Failed payments are visible to user/admin.
- Quotas are granted from successful billing events.
- Manual admin overrides still work.

---

# Milestone 10: Voice Catalogs

## Target

Allow users to manage their own voices while keeping system default voices available.

## Scope

System voices:

- Default voices always available.
- Admin can enable/disable system voices.
- Voices have language and capability metadata.

User voices:

- Add/import user voice
- List user voices
- Delete user voice
- Use user voice for TTS jobs
- Apply new-voice quota
- Store voice assets privately

Admin dashboard:

- View system voices
- View user voices
- Inspect voice usage
- Disable abusive/problematic voices

## Out of scope

- Public voice marketplace
- Sharing voices between users
- Commercial voice licensing automation
- Advanced voice moderation

## Acceptance criteria

- Users can use default voices.
- Users can add private voices if quota allows.
- TTS jobs can select default or user-owned voices.
- Voice assets are access-controlled.
- Admin can inspect and disable voices.

---

# Milestone 11: Exportable Completed Books

## Target

Make completed private books exportable in TwelveReader's existing export format.

## Scope

Completion rules:

- Book metadata complete
- Segments complete
- Synths/audio complete
- Assets attached to the book
- Export manifest valid

Export features:

- Export private book package
- Validate export package
- Rebuild export if assets changed
- Show export readiness in client and admin

Client features:

- Book export status
- Export/download action
- Clear error state if book is not exportable

## Out of scope

- Public publishing
- Explore repository listing
- Private repository auth
- Sharing exports through official repo URLs

## Acceptance criteria

- A completed book can be exported in the TwelveReader format.
- Incomplete books cannot be marked export-ready.
- Export validation errors are visible.
- Export package contains the permanent generated audio assets attached to the book.

---

# Milestone 12: Public Repository Format and Official Public-Domain Catalog

## Target

Build the repository consumption model for Explore without user publishing yet.

## Scope

Repository format:

- Repository manifest
- Book list
- Book metadata
- Asset URLs
- Cover/audio references
- Version/compatibility fields

Official TwelveReader repository:

- Embedded as default Explore repository
- Contains only public-domain books
- Can be removed by the user
- Can be re-added later

Client Explore features:

- Repository list
- Add repository URL
- Remove repository
- Refresh repository
- Browse repository books
- Import/listen from repository depending on existing client model

Repository auth:

- Public repositories only in this milestone.
- No private/basic-auth repositories yet.

## Out of scope

- User public publishing
- User repo identity
- Private repository auth
- Moderation workflows
- DMCA workflows beyond static policy/modal

## Acceptance criteria

- Explore can load the official public-domain repository.
- User can add/remove custom public repository URLs.
- Repository compatibility is validated.
- Broken repositories fail gracefully.
- Official repository is default but not mandatory.

---

# Milestone 13: User Public Repositories

## Target

Allow users to publish completed books to their own public TwelveReader repository.

## Scope

Publishing rules:

- Books are private by default.
- Only completed/exportable books can be made public.
- User must accept copyright/responsibility modal before publishing.
- Public book can be unpublished.
- Public repository only exposes books marked public.

User repo identity:

- Each user has a TwelveReader Repo.
- Repo address uses a DNS-like format that can identify system and user/repo.
- Official hosted server repo identity is distinct from self-hosted repo identity.

Possible repo naming pattern:

- `official.twelvereader://username/main`
- `server-domain.example/@username/main`
- Final scheme can be changed later, but it must separate server/system identity from user/repo identity.

Backend features:

- Public repo endpoint per user
- Public book manifest generation
- Public asset access for published books
- Unpublish flow
- Takedown/remove flow for admin

Client features:

- Publish/unpublish button
- Public/private book state
- Copy repo URL
- Add another user's repo to Explore

## Out of scope

- Private/password-protected repositories
- Organization repositories
- Social discovery
- Recommendations
- Comments/ratings

## Acceptance criteria

- User can publish a completed book.
- User cannot publish incomplete books.
- Published books appear in that user's repo feed.
- Other users can add that repo URL in Explore.
- Unpublished books disappear from public repo output.
- Admin can remove public content if required.

---

# Milestone 14: OAuth for Mobile Platforms

## Target

Add mobile-native account providers after the core SaaS flow works.

## Scope

Authentication providers:

- Google Play / Google account login
- Apple / Game Center-compatible login path where technically appropriate
- Account linking to existing email magic-link accounts
- Provider token verification
- Provider-specific session creation

Client features:

- Continue with Google
- Continue with Apple where supported
- Link/unlink provider if safe
- Login error handling

## Out of scope

- In-app purchases
- Organization accounts
- Social graph
- Game Center-specific gameplay features

## Acceptance criteria

- Users can log in with supported native providers.
- Existing email users can link provider login.
- Sessions behave the same regardless of provider.
- Admin can see auth provider metadata without exposing sensitive tokens.

---

# Milestone 15: Private/Authenticated External Repositories

## Target

Allow Explore repositories to be protected by simple authentication.

## Scope

Supported auth options:

- Basic HTTP auth
- Static bearer token/header
- Password-protected repository config stored locally in the client

Client features:

- Add private repo URL
- Enter credentials
- Validate repository
- Refresh private repository
- Remove credentials when repo is removed

Security behavior:

- Do not send private repo credentials to official TwelveReader servers unless explicitly required by design.
- Prefer client-side repository fetching for external private repos if possible.

## Out of scope

- Full OAuth for external repositories
- Repository federation
- Shared team repositories
- Organization support

## Acceptance criteria

- User can add a basic-auth protected repo.
- Credentials are stored safely according to client platform constraints.
- Invalid credentials fail clearly.
- Public repo behavior remains unchanged.

---

# Milestone 16: SaaS Operations Hardening

## Target

Make the hosted TwelveReader service reliable enough for broader production use.

## Scope

Operational hardening:

- Backups
- Restore process
- Database migrations
- Object storage lifecycle rules
- Orphaned asset cleanup
- Monitoring
- Alerts
- Error-rate dashboards
- Queue dashboards
- Deployment status
- Rollback procedure
- Admin audit log review

Security hardening:

- Rate limiting
- Abuse detection
- Upload validation
- Malware scanning if feasible
- Signed URLs for private assets
- Secrets management
- Sensitive log redaction
- Data export/delete support

Legal/compliance basics:

- Terms of Service
- Privacy Policy
- Copyright policy
- DMCA/contact process
- User responsibility modal for uploads/publishing
- Admin takedown flow

## Out of scope

- New product features
- Organizations/workspaces
- Social discovery
- Marketplace features

## Acceptance criteria

- Production backup and restore process is tested.
- Private assets are not publicly accessible without signed URLs or authorization.
- Admin actions are audited.
- Abuse/rate limits are active.
- Legal documents and upload/publish responsibility modals exist.
- Alerts notify maintainers of critical failures.

---

# MVP Scopes

## MVP 0: SaaS Instrumentation MVP

## Purpose

Make the current system measurable and safer to evolve.

## Includes

- `/api/v1` foundation
- Health/server-info endpoints
- Request IDs
- Structured errors
- Feature flags
- Usage metering in shadow mode
- Usage rollups
- Debug/Admin visibility for usage and jobs

## Excludes

- Login
- Billing
- Enforced quotas
- Client account flow

## Exit condition

The current app works as before, but the backend can now report what resources are being consumed.

---

## MVP 1: Quota-Controlled Local SaaS Core

## Purpose

Make the backend behave like a SaaS resource manager before real users and payments are added.

## Includes

- Usage metering
- Daily/weekly/monthly quotas
- Manual quota grants
- Quota denial errors
- Lazy `next N segments` generation
- Job queue visibility
- Admin Dashboard shell with Debug section

## Excludes

- Real accounts
- Stripe
- Public repositories
- OAuth

## Exit condition

The system can control expensive actions through quotas and incremental generation, even if all usage still belongs to a bootstrap/default user.

---

## MVP 2: Account-Based Private Library

## Purpose

Turn TwelveReader into a private, account-based hosted service.

## Includes

- Users
- Roles: user/admin
- Sessions and refresh tokens
- Email magic-link login
- Admin Dashboard access control
- User-owned books
- User-owned jobs/assets/progress
- Client server selection
- Client login/logout
- Private library
- Upload/import
- Book detail/status
- Playback/reading progress sync
- Delete/manage books
- Quota display and quota errors

## Excludes

- Stripe
- Public publishing
- Explore repository publishing
- Mobile-native OAuth
- User voice catalog

## Exit condition

A user can select a server, log in, upload/import a book, generate/listen incrementally, sync progress, and manage their private library under quota limits.

---

## MVP 3: Internal Billing and Voice Catalog MVP

## Purpose

Add the SaaS business model internally before collecting real payments.

## Includes

- Plans
- Subscriptions as internal records
- Credit balances
- Credit transactions
- Quota grants from plans/credits
- Admin plan assignment
- Admin credit grants
- Default system voices
- User private voices
- Voice quotas

## Excludes

- Stripe
- App store payments
- Public voice marketplace
- Public repositories

## Exit condition

Admins can simulate real paid plans and credit-based usage. Users can consume resources according to plan/credit rules and use default or private voices.

---

## MVP 4: Paid Hosted SaaS MVP

## Purpose

Connect the internal billing system to real money.

## Includes

- Stripe customer creation
- Credit purchase checkout
- Subscription checkout
- Stripe webhooks
- Invoices
- Payment failure handling
- Plan upgrade/downgrade
- Billing page in client
- Billing management in Admin Dashboard

## Excludes

- App Store / Play Store purchases
- Organizations/workspaces
- Repository publishing

## Exit condition

A hosted user can buy credits or subscribe, and the system automatically grants quotas based on successful payments.

---

## MVP 5: Explore and Public Repository MVP

## Purpose

Deliver the open audiobook repository model.

## Includes

- Public repository format
- Official TwelveReader public-domain repository
- Client Explore repository list
- Add/remove custom public repositories
- Completed book export validation
- User public repositories
- Publish/unpublish completed books
- Public/private book state
- Copyright responsibility modal
- Admin takedown flow

## Excludes

- Private/password-protected repositories
- Organization repositories
- Social discovery
- Recommendations

## Exit condition

Users can browse the official public-domain repository, add other public repositories, and publish completed books to their own public repository.

---

## MVP 6: Mobile Auth and Production Hardening MVP

## Purpose

Make the product ready for wider mobile distribution and production operations.

## Includes

- Google login
- Apple login where applicable
- Account linking
- Signed URLs for private assets
- Rate limiting
- Upload validation
- Backup/restore
- Monitoring/alerts
- Audit log review tools
- Terms/Privacy/Copyright/DMCA documents
- Data export/delete
- Production deployment and rollback process

## Excludes

- Organizations/workspaces
- Marketplace features
- Private external repositories unless prioritized separately

## Exit condition

TwelveReader is ready for broader hosted beta usage with mobile-native login and production-grade operational controls.

---

# Deferred / Later Scope

These are intentionally not part of the early SaaS milestones:

- Organizations/workspaces
- Team libraries
- Private repository authentication
- Public social discovery
- Recommendations
- Comments/ratings
- Voice marketplace
- App Store / Play Store in-app purchases
- Advanced copyright automation
- Publisher/creator portals
- Federation between TwelveReader servers

---

# Recommended Development Order

1. Milestone 0: SaaS Readiness Baseline
2. Milestone 1: Usage Metering Ledger, Shadow Mode
3. Milestone 2: Quota Engine, Non-Billing Enforcement
4. Milestone 3: Lazy Generation Pipeline
5. Milestone 4: Admin Dashboard Shell
6. Milestone 5: Accounts and Sessions
7. Milestone 6: Client Server Selection and Login
8. Milestone 7: Private User Library
9. Milestone 8: Plans, Credits, and Subscriptions Without Stripe
10. Milestone 9: Stripe Billing Integration
11. Milestone 10: Voice Catalogs
12. Milestone 11: Exportable Completed Books
13. Milestone 12: Public Repository Format and Official Public-Domain Catalog
14. Milestone 13: User Public Repositories
15. Milestone 14: OAuth for Mobile Platforms
16. Milestone 15: Private/Authenticated External Repositories
17. Milestone 16: SaaS Operations Hardening

This order keeps each milestone focused. Earlier milestones create infrastructure and control points. Later milestones add product surface area. Future milestones can change without invalidating the already-finished foundation.
