# 🛡️ Quest Log (The Forge)
An offline-first, high-performance background engine and task management system built in Go. Operating locally on home lab infrastructure, it coordinates task-based urgency lifecycles and gamified behavioral analytics for multiple isolated user profiles.

This repository implements a resilient, ADHD-friendly execution framework designed to bridge the gap between long-term high-level goals (such as system design mastery, literature production, and physical conditioning) and daily executive function.

## 🏗️ Core System Architecture
Quest Log functions as a modular web application. By replacing traditional time-blocking models with **Task-Based Urgency** and server-validated **Dopamine-Linked Rewards**, this tool maintains a structured workspace to balance shared operations and personal micro-milestones.

```
                  ┌────────────────────────────────────────┐
                  │          static/css/style.css          │
                  │   (Low-Contrast Migraine-Safe CSS)     │
                  └───────────────────┬────────────────────┘
                                      │ (Token Extraction)
                                      ▼
┌────────────────────────────────────────────────────────────────────────┐
│                          internal/web/ (Handlers)                      │
│   Parses Inbound Payloads • Form Sanitization • Template Composition   │
└───────────────┬────────────────────────────────────────┬───────────────┘
                │                                        │
                │ (Transactional Writes)                 │ (Evaluates State)
                ▼                                        ▼
┌──────────────────────────────┐        ┌────────────────────────────────┐
│ internal/repository/ (DAO)   │        │     internal/database/         │
│   Executes Ledger Operations │        │   Connection Pool Enclosure    │
└───────────────┬──────────────┘        └────────────────┬───────────────┘
                │                                        │
                └────────────────────┐ ┌─────────────────┘
                                     ▼ ▼
				┌────────────────────────────────────────┐
				│            data/quests.db              │
				│       (SQLite3 Engine • WAL Mode)      │
				└────────────────────────────────────────┘
```

## ⚡ Task Lifecycle Processing & Telemetry Ingestion
```
[User Action Input]
 │  POST /quests/create (Form Ingestion Gateway)
 ▼
[Ghost Guard Layer]
 │  String whitespace sanitization & structural check gates
 ▼
[Type Evaluation Fork]
 │
 ├───► (One-Time Bounty) ──► Insert directly into active ledger array
 │
 ├───► (Repeating Loop) ───► Calculate post-completion custom interval gap
 │
 └───► (Static Weekly) ────► Bind reset vector to target day-of-week integer
 ▼
[Hard-Coded Economy Validation]
 │  Parse incoming tier index token (1, 2, or 3)
 │  Map signature reward currency programmatically:
 │    - Tier 1: 🪙 Coin (1 XP)
 │    - Tier 2: 💰 Moneybag (5 XP)
 │    - Tier 3: 👑 Crown (10 XP)
 ▼
[Database Execution Pool]
 │  Commit parsed record down to database.DB connection context
 ▼
[State Complete Signal]
 └─► POST /quests/complete ──► Log to Immutable Chronicle Ledger (`quest_completions`)
```

## 🎛️ Core Philosophy & Engineering Constraints
1. **Low-Contrast Visual Architecture:** Built explicitly around a custom-tuned, light-absorbing dark mode canvas (`#12161F` and `#1E2533`). By abandoning high-contrast white text flashes and intense neon saturation, the system layout drastically limits cognitive eye strain and prevents visual vibration during barometric pressure swings.
2. **The Hard-Coded Economy:** Eliminates arbitrary point value inflation. Task rewards are strictly compressed to static server-evaluated integers ($1$, $5$, $10$), ensuring long-term ledger consistency.
3. **Strategic Momentum Triage:** Implements an immediate frontend filter toggle ("Momentum Mode"). When active, the query engine limits database scanning outputs exclusively to `is_non_negotiable` tasks, lowering the interface cognitive load down to zero during tight windows.
4. **Structured DevOps Telemetry:** Employs explicit, machine-readable console visual tracking wrappers (`[INIT]`, `[SECURE]`, `[OK]`, `[ERROR]`, `[REALTIME]`) to ensure clean terminal observation under container runtimes.
5. **Idempotent Storage Infrastructure:** Combines strict relational SQLite constraint safety layers with a transactional background checkpoint mechanism to guarantee file persistence inside Docker volume boundaries.

## 🛠️ Tech Stack & Runtime
- **Language Runtime:** Go 1.24+ (Native structured templates, type-safe error propagation, and context-aware database bindings)
- **Database Engine:** SQLite 3 via `github.com/mattn/go-sqlite3` operating under Write-Ahead Logging (`WAL` mode)
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
- [x] **Standard Package Decoupling:** Re-architect monolithic, single-file internal spaces into clean, scoped domain package boundaries (`database`, `repository`, `web`).
- [x] **Directory Relocation:** Shift the system execution boot layout to `cmd/main.go` to conform to standard Go project layouts.
- [x] **Terminology Migration:** Wipe away legacy agricultural labels across all components, updating references to **Bounty Board**, **The Forge**, and **The Chronicle**.

#### **Phase 4: Focus Telemetry & Visual Refactor (COMPLETED)**
- [x] **Muted Obsidian Theme:** Deploy a low-contrast, custom dark mode interface across all layout files to prevent cognitive fatigue and eye strain.
- [x] **Active View Triage Toggle:** Connect the frontend **"Momentum Mode"** switch to a URL query parameter filtration mechanism that hides standard targets under high-pressure scenarios.
- [x] **Cache Shielding:** Apply version parameter strings (`style.css?v=3.0.1`) to elements to cleanly bypass aggressive local browser stylesheet caching bugs.

#### **Phase 5: Storage Optimization & Maintenance (COMPLETED)**
- [x] **Engine Hygiene:** Automated `db.Exec("VACUUM")` database compaction routines to claim unallocated disk sectors after data purging.
- [x] **Data Pruning Ledger:** Background utility to safely purge historical logs from `quest_completions` older than a defined retention window (e.g., 14 days) to permanently limit SQLite file bloat.
- [x] **Automated Disaster Recovery:** Lightweight cron routine to create timestamped, compressed backups (`tar.gz`) of the SQLite database file, keeping a rolling window of copies stored safely outside the live container volume.
- [x] **Graceful Teardown Loop:** Configure system lifecycle interrupt interceptors (`SIGINT`, `SIGTERM`) to force connection pool checkpoints, ensuring SQLite cleanly collapses WAL files back to disk on container exits.

#### **Phase 6: Analytics Ledger & Interface Sorting (COMPLETED)**
- [x] **The Chronicle Summary Engine:** Build an aggregation pipeline that runs every Sunday evening to compile a weekly operational report tracking precise execution frequencies and task type breakdowns.
- [x] **Triage Layout Sorting:** Refactor the frontend query logic to sort active contracts by matching **Category Grouping** arrays instead of default table insert sequence, visually grouping identical real-world contexts together.

#### **Phase 7: The Ingestion Bridge (PLANNED)**
- [ ] **Automated Seeding Engine:** Build a file-based JSON bulk-importer for fast profile onboarding and contract minting.
- [ ] **Headless API Exposure:** Secure headless endpoints to pipe live task analytics straight into Obsidian Dataview notebooks via local JSON payloads.

#### **Phase 8: Multi-User Architecture & Personalization (PLANNED)**
- [ ] **Session Authentication Layer:** Implement a lightweight, secure session state manager to protect individual dashboard profiles.
- [ ] **Dynamic Interface Swapping:** Finalize native design token flags to support clean switching to alternative styles (like a future `style_light.css`) seamlessly from the web interface.

#### **Phase 9: Advanced Environmental Integrations (PLANNED)**
- [ ] **Pre-Flight Gear Check Lock:** Design conditional contract gating where active bounties remain locked out in the UI until explicit pre-requisite check toggles are verified.
- [ ] **Atmospheric Automation Trigger:** Integrate local background telemetry daemons with your custom `Barometric Guard` API to automatically signal the application to suggest turning on **Momentum Mode** during severe pressure drops.