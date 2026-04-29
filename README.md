# 🛡️ Quest Log (The Forge)
A Go-based **Relational Task Engine** and CRUD application running on the **Milford Node**. This project implements a gamified, ADHD-friendly productivity framework designed to bridge the gap between high-level creative goals and daily executive function.

## 🧠 Project Overview
Quest Log acts as the "Command Center" for the Milford Node ecosystem. By replacing traditional time-blocking with **Task-Based Urgency** and **Dopamine-Linked Rewards**, this tool provides a structured environment to manage shared household operations and individual professional growth.

### 🚀 Key Features
- **The Logic Trio (Defensive Engineering):**
    - **Ghost Guard:** Server-side validation and atomic `sql.Transaction` logic to prevent partial writes and ensure data sanitization.
    - **Hard-Coded Economy:** A "Number Compressed" XP system ($1, 5, 10$) to prevent reward-inflation and maintain consistent effort-tracking.
    - **Priority Shield:** A high-visibility triage layer that floats "Non-Negotiable" quests to the top of the stack.
- **Automated Quest Lifecycle:** A background **Master Spawner** (via `robfig/cron`) that orchestrates Daily resets and Interval-based recurrence using Julian Day delta calculations.
- **The Weekly Corral:** A dedicated historical ledger (`quest_completions`) that decouples task state from accomplishment tracking. It provides a visual "Pile of Wins" that persists even after the "Pasture" is cleared.
- **Idempotent Infrastructure:** A single-binary deployment model using `go:embed` to bake the SQL schema directly into the application, ensuring a portable and consistent environment.

## 🛠️ Tech Stack
- **Language:** Go (Golang) 1.2x
- **Scheduling:** `robfig/cron/v3`
- **Database:** SQLite3 (WAL Mode, Relational Schema with `CHECK` constraints)
- **Frontend:** HTML5 | CSS3 Flexbox | Go Templates

## 🗄️ Database Architecture

| **Entity** | **Purpose** | **Key Logic** |
|--- |--- |--- |
| **Quests** | The Core Unit | Difficulty mapping (Duck/Sheep/Cow) |
| **Users** | Persistence Layer | Tracks `dopamine_streak` and user-specific stats |
| **Completions** | The Ledger | Immutable history of XP awarded and completion timestamps |
| **Categories** | Visual Context | Hex-code mapping for `quest-accent-bar` |
| **Gear Checks** | Blocking Unit | Boolean check to unlock Quest "Complete" status |

## 🚀 Execution Roadmap

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
- [x] **Cold Start Resilience:** Added idempotent startup checks to prevent missed cycles during downtime.

#### **Phase 4: Momentum & Telemetry (IN PROGRESS)**
- [ ] **The "Bad Brain Day" Momentum Filter:** UI-triggered backend toggle to restrict views to `is_non_negotiable` tasks.
- [ ] **The Gear Check Lock:** Pre-flight validation step where quests remain locked until `gear_checks` are toggled.

#### **Phase 5: The Ingestion Bridge & Telemetry**
- [ ] **Automated Seeding:** Building a JSON bulk-importer for rapid task generation.
- [ ] **Atmospheric Trigger:** Integrating with the `Barometric Pressure Warning` daemon to suggest "Low-Energy Mode" during weather shifts.
- [ ] **API Exposure:** Finalizing the headless endpoint for Obsidian Dataview visualization.