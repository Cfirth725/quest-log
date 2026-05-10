
# 🛡️ Quest Log (The Forge)

A Go-based **Relational Task Engine** and CRUD application running on the **Milford Node**. This project implements a gamified, ADHD-friendly productivity framework designed to bridge the gap between high-level creative goals and daily executive function.

## Project Overview

Quest Log acts as the "Command Center" for the Milford Node ecosystem. By replacing traditional time-blocking with **Task-Based Urgency** and **Dopamine-Linked Rewards**, this tool provides a structured environment to manage shared household operations and individual professional growth.

### Key Features

- **The Logic Trio (Defensive Engineering):**
    - **Ghost Guard:** Server-side validation and atomic `sql.Transaction` logic to prevent partial writes and ensure data sanitization.
    - **Hard-Coded Economy:** A "Number Compressed" XP system ($1, 5, 10$) to prevent reward-inflation and maintain consistent effort-tracking.
    - **Priority Shield:** A high-visibility triage layer that floats "Non-Negotiable" quests to the top of the stack.
- **Automated Quest Lifecycle:** A background **Master Spawner** (via `robfig/cron`) that orchestrates Daily resets and Interval-based recurrence using Julian Day delta calculations.
- **The Weekly Corral:** A dedicated historical ledger (`quest_completions`) that decouples task state from accomplishment tracking. It provides a visual "Pile of Wins" that persists even after the "Pasture" is cleared.
- **Idempotent Infrastructure:** A single-binary deployment model using `go:embed` to bake the SQL schema directly into the application, ensuring a portable and consistent environment.
- **State-Machine Management:** Integrated "Pivot" logic allowing users to downgrade repeating tasks to **One-Time** quests dynamically, adapting to fluctuating cognitive energy levels.

### Infrastructure & Deployment

- The Quest Log is designed to run as a resilient "Always-On" service on the **Milford Node** (Raspberry Pi 3B).
- **Container Orchestration:** Managed via Docker Compose with directory-level volume persistence to ensure SQLite journal files (`-wal` and `-shm`) survive container lifecycles.
- **Precision Timekeeping:** Uses `_loc=Local` and `parseTime=true` DSN parameters to align the Go application layer with the physical hardware clock, ensuring accurate "Weekly Win" telemetry.
- **Multi-Stage Builds:** A optimized Debian-based Dockerfile that balances a heavy GCC build environment (for CGO/SQLite) with a slim production runtime.

## Tech Stack

- **Language:** Go (Golang) 1.2x
- **Scheduling:** `robfig/cron/v3`
- **Database:** SQLite3 (WAL Mode, Relational Schema with `CHECK` constraints)
- **Frontend:** HTML5 | CSS3 (Modular Token-Based Design) | Go Templates

## Database Architecture

|**Entity**|**Purpose**|**Key Logic**|
|---|---|---|
|**Quests**|The Core Unit|Difficulty mapping (Duck/Sheep/Cow)|
|**Users**|Persistence Layer|Tracks `dopamine_streak` and user-specific stats|
|**Completions**|The Ledger|Immutable history of XP awarded|
|**Categories**|Visual Context|Hex-code mapping for `quest-accent-bar`|
|**Archive**|Soft-Delete Layer|Implements `deleted_at` for non-destructive task retirement|

## Execution Roadmap

#### **Phase 1: The Core Foundation (COMPLETE)**
- [x] Establish Go-SQLite connectivity with unified `RenderTemplate` logic.
- [x] Implement **The Logic Trio**: Ghost Guard, Hard-Coded Economy, and Priority Shield.
- [x] Design dynamic Category loading with Hex-code visual mapping.

#### **Phase 2: Transition & Reward Engine (COMPLETE)**
- [x] **Atomic Transactions:** Finalized `CompleteQuest` logic to ensure data integrity.
- [x] **The Reward Ledger:** Implementation of the **Weekly Corral** to track XP independently.
- [x] **Portable Schema:** Integrated `go:embed` for idempotent database initialization.

#### **Phase 3: Executive Function Hardening (COMPLETE)**
- [x] **Modular Architecture:** Refactored codebase into a professional multi-file structure.
- [x] **Master Spawner Logic:** Implemented Cron-based resets for `Daily` and `Repeating` quests.
- [x] **Design System Refactor:** Migrated to a centralized, token-based CSS architecture for better maintainability.

#### **Phase 4: Momentum & Telemetry (IN PROGRESS)**
- [x] **The "Bad Brain Day" Momentum Filter:** UI-triggered backend toggle to restrict views to `is_non_negotiable` tasks.
- [x] **Soft-Delete Safety Net:** Implemented non-destructive deletion via `deleted_at` timestamps.
- [x] **Resilient Data Plumbing:** - Implemented **Graceful Shutdown** orchestration to handle `SIGINT`/`SIGTERM`, ensuring `db.Close()` merges SQLite WAL files before container exit.
	- [x] Synchronized the **Milford Node** clock to local time, anchoring automated resets (Dailies) to a 4:00 AM local window rather than UTC midnight.
- [ ] **The Gear Check Lock:** Pre-flight validation step where quests remain locked until `gear_checks` are toggled.
- [ ] **Atmospheric Trigger:** Integrating with the `Barometric Pressure Warning` daemon to suggest "Low-Energy Mode" during weather shifts.

#### **Phase 5: The Ingestion Bridge**
- [ ] **Automated Seeding:** Building a JSON bulk-importer for rapid task generation.
- [ ] **API Exposure:** Finalizing the headless endpoint for Obsidian Dataview visualization.

#### **Phase 6: Multi-User Plan**
- [ ] **Auth Layer:** Implementing secure session management and user-specific pasture views.
