# TwelveReader Manifest

## Vision

TwelveReader is an open-source AI audiobook platform that can run as both a hosted SaaS and a self-hosted server. Users download the mobile-first client, choose the official server or a custom server before login, and keep a private account-based audiobook library. The hosted service adds managed TTS generation, quotas, billing, admin/support tooling, public-domain official repositories, and user-published TwelveReader repositories.

The project should stay useful without the official hosted service. Self-hosted TwelveReader servers are first-class, and the client must support changing the server before login.

## Product Model

- Main unit: individual user account.
- No organizations/workspaces in the current SaaS scope.
- Users own isolated private libraries; no deduplication or shared internal objects between users.
- Users can upload/import books privately by default.
- Completed books can be exported in TwelveReader's package format.
- Completed/exportable private books can later be published to the user's public TwelveReader repository.
- The official TwelveReader organization hosts only public-domain books.
- Explore works as a collection of repositories: official public-domain repo by default, plus user-added repositories.
- Users may remove the official repository and add any compatible public repository.
- User voice catalogs are supported later: system voices remain available, users can add private voices within quota.

## SaaS Operating Model

TwelveReader SaaS is built around usage metering and quota enforcement before payment integration:

- Storage quota
- Segment / LLM token quota
- TTS synthesis minutes
- New voice quota
- Audio listen minutes
- Daily, weekly, and monthly windows
- Manual/admin quota grants first
- Internal plans and credits second
- Stripe integration after internal billing is stable

The generation pipeline should move from whole-book eager segmentation/synthesis to a fair `next N segments` model, so users consume quota only as they actually read/listen and prefetch within configured limits.

## Admin Dashboard Direction

The current Debug Dashboard becomes a section of the Admin Dashboard.

Admin Dashboard sections:

- Overview
- Users
- Books
- Jobs
- Voices / Models
- Storage
- Billing / Usage
- Support
- Debug
- Audit Log

The dashboard is for managing, inspecting, debugging, support operations, subscriptions, quotas, users, books, jobs, and system health. Access is role-gated.

## Client Direction

The TwelveReader client becomes mobile-first and account-aware while continuing to use the current web-native technology stack.

Required direction:

- Server selection before login
- Official server as default
- Custom/self-hosted server support
- Email magic-link login first
- Mobile OAuth providers later
- Private user library
- Upload/import
- Book status and job status
- Playback/reading progress sync
- Quota/usage visibility
- Explore repositories
- Publishing completed books later

## Legal / Copyright Direction

Early SaaS scope uses a clear responsibility modal: users are responsible for uploaded and shared material. If authorities or rightsholders contact TwelveReader about copyrighted public material, the material can be removed from the user's repository. Full copyright automation is deferred.

## Canonical SaaS Planning Document

The full SaaS vision, milestone breakdown, and MVP scopes live in:

- [docs/SAAS_MANIFEST.md](docs/SAAS_MANIFEST.md)

## Reference Documents

- SaaS manifest and roadmap: [docs/SAAS_MANIFEST.md](docs/SAAS_MANIFEST.md)
- High-level architecture: [SystemDesign.md](SystemDesign.md)
- API plan: [API.md](API.md)
- Data and packaging formats: [DataFormats.md](DataFormats.md)
- Delivery milestones: [Milestones.md](Milestones.md)
- Current task index: [TASKS.md](TASKS.md)
