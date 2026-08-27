# 🛡️ Quest Log (The Forge)
An offline-first, high-performance background engine and task management system built in Go. Operating locally on home lab infrastructure, it coordinates task-based urgency lifecycles and gamified behavioral analytics for multiple isolated user profiles and shared household environments.

This repository implements a resilient, ADHD-friendly execution framework designed to bridge the gap between long-term high-level goals (such as system design mastery, literature production, and physical conditioning) and daily executive function.

## 🏗️ Core System Architecture
Quest Log functions as a modular web application and headless API server. By replacing traditional time-blocking models with **Task-Based Urgency** and server-validated **Dopamine-Linked Rewards**, this tool maintains a structured workspace to balance shared operations and personal micro-milestones.

```
                     ┌────────────────────────────────────────┐
                     │         static/css/style.css           │
                     │   (Low-Contrast Migraine-Safe CSS)     │
                     └───────────────────┬────────────────────┘
                                         │ (Token Extraction)
                                         ▼
┌────────────────────────────────────────────────────────────────────────┐
│                        internal/web/ (Handlers)                        │
│   Parses Inbound Payloads • Form Sanitization • Template Composition   │
└─────────┬──────────────────────────────┬───────────────────────┬───────┘
          │                              │                       │
          │ (Transactional Writes)       │ (Evaluates State)     │ (Session Middleware)
          ▼                              ▼                       ▼
┌──────────────────────────┐   ┌───────────────────┐   ┌──────────────────────────┐
│ internal/repository/ DAO │   │ internal/database │   │   internal/middleware    │
│ Executes Ledger Ops      │   │ Conn Pool Encl    │   │  SessionAuth & API Guard │
└─────────┬────────────────┘   └─────────┬─────────┘   └─────────┬────────────────┘
          │                              │                       │
          └────────────────────┐         │           ┌───────────┘
                               ▼         ▼           ▼
                     ┌────────────────────────────────────────┐
                     │            data/quests.db              │
                     │      (SQLite3 Engine • WAL Mode)       │
                     └────────────────────────────────────────┘
```

## ⚡ Task Lifecycle Processing & Telemetry Ingestion
```
[Inbound Request Gateway]
│ ├──► POST /login (PIN Authentication Gateway)
│ ├──► POST /quests/create (Form Ingestion Gateway)
│ ├──► POST /api/v1/quests/import (Bulk Scriptorium Import)
│ └──► GET /api/v1/telemetry (Headless API Entry)
▼
[Security & Middleware Layer]
│ ├──► (API Requests) ──► Validate X-API-Key / Bearer Token against QUESTLOG_API_KEY
│ └──► (Web Requests) ──► SessionAuth checks HttpOnly cookie ──► Injects User into r.Context()
▼
[Owner Scope & Filter Gate]
│ Scopes task visibility: owner_id IN (active_user_id, 0 [Household Shared])
│ ├──► (Momentum Mode) ─► Restricts query execution to is_non_negotiable priority tasks
│ └──► (Scriptorium) ───► Case-insensitive owner mapping ("User", "Household", ID)
▼
[Type Evaluation Fork]
│ ├───► (One-Time Bounty) ──► Insert directly into active ledger array
│ ├───► (Repeating Loop) ───► Calculate post-completion custom interval gap
│ └───► (Static Weekly) ────► Bind reset vector to target day-of-week integer
│ (Evaluated daily by background cron at 04:03 AM EDT)
▼
[Hard-Coded Economy Validation]
│ Parse incoming tier index token (1, 2, or 3)
│ Map signature reward currency programmatically:
│ - Tier 1: 🪙 Coin (1 XP)
│ - Tier 2: 💰 Moneybag (5 XP)
│ - Tier 3: 👑 Crown (10 XP)
▼
[Database Execution Pool]
│ Commit parsed record down to database.DB connection context
▼
[State Complete / Observability Signal]
├─► POST /quests/complete ──► Log to Immutable Chronicle Ledger (`quest_completions`)
└─► GET /api/v1/telemetry ─► Aggregate active workload, daily XP, & taxonomy matrix JSON
```

## ✨ Core Philosophy & Engineering Constraints
1. **Low-Contrast Visual Architecture:** Built explicitly around a custom-tuned, light-absorbing dark mode canvas (`#12161F` and `#1E2533`) paired with a warm, low-glare Parchment light mode (`#F4EFE6`). The system layout minimizes cognitive fatigue and eliminates harsh contrast flashes during state transitions.
2. **Multi-User & Shared Household Scoping:** Fully supports isolated participant profiles alongside shared `0 (Household)` contracts. Tasks and category taxonomies dynamically filter to show personal bounties alongside shared household objectives (`owner_id IN (?, 0)`).
3. **Session-Based Security Architecture:** Lightweight PIN authentication gateway using salted `bcrypt` password hashing paired with cryptographically secure 32-byte session tokens stored in SQLite. Tokens are delivered via secure `HttpOnly` browser cookies and injected into request contexts (`r.Context()`) via middleware.
4. **The Hard-Coded Economy:** Eliminates arbitrary point value inflation. Task rewards are strictly compressed to static server-evaluated integers ($1$, $5$, $10$), ensuring long-term ledger consistency.
5. **Strategic Momentum Triage:** Implements an immediate frontend filter toggle ("Momentum Mode"). When active, the query engine limits database scanning outputs exclusively to `is_non_negotiable` tasks, lowering the interface cognitive load down to zero during tight windows.
6. **Flex-Owner Bulk Ingestion Engine:** The Arcane Scriptorium accepts raw JSON manifests featuring flexible owner specifications—resolving integer IDs, case-insensitive user names, or shared keywords (`"Household"`, `"Shared"`) seamlessly during batch imports.
7. **Structured DevOps Telemetry:** Employs explicit, machine-readable console visual tracking wrappers (`[INIT]`, `[SECURE]`, `[OK]`, `[ERROR]`, `[REALTIME]`) to ensure clean terminal observation under container runtimes.

## 🛠️ Tech Stack & Runtime
- **Language Runtime:** Go 1.24+ (Native structured templates, type-safe error propagation, Go 1.22+ enhanced `net/http` routing, and context-aware database bindings)
- **Database Engine:** SQLite 3 via `github.com/mattn/go-sqlite3` operating under Write-Ahead Logging (`WAL` mode)
- **Security & Authentication:** `golang.org/x/crypto/bcrypt`, `crypto/rand` session generation, `HttpOnly` cookies
- **Design System:** Vanilla CSS3 (Centralized Design Tokens)
- **Orchestration Matrix:** Docker Multi-stage Linux Build

## 🗺️ Execution Roadmap
#### **Phase 1: The Core Foundation (COMPLETED)**
- [x] Establish Go-SQLite connectivity with decoupled, unified `RenderTemplate` compilation utilities.
- [x] Implement **The Logic Trio**: Ghost Guard input validation, Hard-Coded Economy scaling values, and Critical Path Priority Shields.
- [x] Design dynamic Category loading matrices with custom Hex-code visual mapping hooks.

#### **Phase 2: Transition & Reward Ledger (COMPLETED)**
- [x] **Atomic Transactions:** Secure the transactional logic loop inside `CompleteQuest` to eliminate multi-write database faults.
- [x] **The Chronicle Base:** Construct the immutable historical database table `quest_completions` to log completion data and metrics independently.
- [x] **Timezone Alignment:** Bind data collection and query scopes to local hardware clock configurations (`_loc=Local`) to stabilize automated window resets.

#### **Phase 3: Domain Package Hardening (COMPLETED)**
- [x] **Standard Package Decoupling:** Re-architect monolithic, single-file internal spaces into clean, scoped domain package boundaries (`database`, `repository`, `web`, `middleware`).
- [x] **Directory Relocation:** Shift the system execution boot layout to `cmd/main.go` to conform to standard Go project layouts.
- [x] **Terminology Migration:** Wipe away legacy agricultural labels across all components, updating references to **Bounty Board**, **The Forge**, and **The Chronicle**.

#### **Phase 4: Focus Telemetry & Visual Refactor (COMPLETED)**
- [x] **Muted Obsidian Theme:** Deploy a low-contrast, custom dark mode interface across all layout files to prevent cognitive fatigue and eye strain.
- [x] **Active View Triage Toggle:** Connect the frontend **"Momentum Mode"** switch to a URL query parameter filtration mechanism that hides standard targets under high-pressure scenarios.
- [x] **Cache Shielding:** Apply version parameter strings (`style.css?v=3.1.4`) to elements to cleanly bypass aggressive local browser stylesheet caching bugs.

#### **Phase 5: Storage Optimization & Maintenance (COMPLETED)**
- [x] **Engine Hygiene:** Automated `db.Exec("VACUUM")` database compaction routines to claim unallocated disk sectors after data purging. 
- [x] **Data Pruning Ledger:** Background utility to safely purge historical logs from `quest_completions` older than a defined retention window (e.g., 14 days) to permanently limit SQLite file bloat.
- [x] **Automated Disaster Recovery:** Lightweight cron routine to create timestamped, compressed backups (`tar.gz`) of the SQLite database file, keeping a rolling window of copies stored safely outside the live container volume.
- [x] **Graceful Teardown Loop:** Configure system lifecycle interrupt interceptors (`SIGINT`, `SIGTERM`) to force connection pool checkpoints, ensuring SQLite cleanly collapses WAL files back to disk on container exits.

#### **Phase 6: Analytics Ledger & Interface Sorting (COMPLETED)**
- [x] **The Chronicle Summary Engine:** Build an aggregation pipeline anchored to Sunday 00:00 AM local time to compile weekly operational reports, completion counts, and habit loop frequencies.
- [x] **Triage Layout Sorting:** Refactor the query logic to sort active contracts by Category Grouping arrays, status flags, and Priority Shields instead of raw insert ID order.

#### **Phase 7: The Ingestion Bridge & Headless API (COMPLETED)**
- [x] **The Arcane Scriptorium:** Build a file-based JSON bulk-importer (`/scriptorium`) with real-time pre-flight category mapping analysis and batch transactional ingestion.
- [x] **Headless Telemetry Endpoint:** Secure `/api/v1/telemetry` with zero-trust API key middleware to export live workload counts, daily XP disbursements, and category breakdowns for external consumption.

#### **Phase 8: Multi-User Architecture & Personalization (COMPLETED)**
- [x] **Session Authentication Layer:** Implement a lightweight, secure session state manager (`sessions` table + `SessionAuth` middleware) to protect individual dashboard profiles using PIN authentication.
- [x] **User-Scoped Query Scoping:** Update repository queries to scope task visibility (`owner_id IN (?, 0)`), category management, and XP completion gains directly to the active session user.
- [x] **Flexible Ingestion Owner Resolution:** Enable case-insensitive owner name matching inside the Arcane Scriptorium bulk ingestion pipeline.
- [x] **Character UI Badge & Utility Island:** Deploy a unified top-level utility control badge (Theme switcher, User identity badge, and Logout portal).
- [x] **Dynamic Interface Swapping:** Implement an instant, anti-flash theme engine supporting both Muted Obsidian (Dark) and warm Parchment (Light) palettes persisted in `localStorage`.

#### **Phase 9: Workflow Automation & Advanced Capabilities (PLANNED)**
- [ ] **Pre-Flight Subtask Checklist Lock:** Implement prerequisite checklist steps within complex bounties, locking completion until all itemized subtasks are marked ready.
- [ ] **Export & Data Portability Suite:** Provide single-click JSON/CSV export endpoints for personal Chronicle histories and active quest boards to facilitate external analytics.
- [ ] **Custom Webhook Dispatcher:** Add an event-driven webhook subsystem to emit JSON payloads (e.g., Discord/Gotify notifications) when high-tier bounties or weekly targets are achieved.
