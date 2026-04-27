# 🛡️ Quest Log (The Forge)
A Go-based **Relational Task Engine** and CRUD application running on the **Milford Node**. This project implements a gamified, ADHD-friendly productivity framework designed to bridge the gap between high-level creative goals (Novel Publication) and daily executive function.

## 🧠 Project Overview
Quest Log acts as the "Command Center" for the Milford Node ecosystem. By replacing traditional time-blocking with **Task-Based Urgency** and **Dopamine-Linked Rewards**, this tool provides a structured environment to manage shared household operations and individual professional growth.

### 🚀 Key Features
- **The Logic Trio (Defensive Engineering):**
    - **Ghost Guard:** Server-side validation and atomic `sql.Transaction` logic to prevent partial writes and ensure data sanitization.
    - **Hard-Coded Economy:** A "Number Compressed" XP system ($1, 5, 10$) to prevent reward-inflation and maintain consistent effort-tracking.
    - **Priority Shield:** A high-visibility triage layer that floats "Non-Negotiable" quests to the top of the stack.
- **The Weekly Corral:** A dedicated historical ledger (`quest_completions`) that decouples task state from accomplishment tracking. It provides a visual "Pile of Wins" that persists even after the "Pasture" is cleared.
- **Idempotent Infrastructure:** A single-binary deployment model using `go:embed` to bake the SQL schema directly into the application, ensuring a portable and consistent environment.
- **Dopamine Counter:** A dual-track reward system that tracks pure XP for the "Corral" while maintaining a `dopamine_streak` counter in the user profile to track long-term momentum.

## 🛠️ Tech Stack
- **Language:** Go (Golang) 1.2x
- **Database:** SQLite3 (Relational Schema with `CHECK` constraints)
- **Frontend:** HTML5 | CSS3 Flexbox | Go Templates
- **Infrastructure:** Raspberry Pi 3 Model B | Docker | Portainer

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
- [x] Implement **The Logic Trio**: Ghost Guard (validation), Hard-Coded Economy (XP constants), and Priority Shield (sorting).
- [x] Design dynamic Category loading with Hex-code visual mapping.

#### **Phase 2: Transition & Reward Engine (COMPLETE)**
- [x] **Atomic Transactions:** Finalized `CompleteQuest` logic in the repository to ensure data integrity.
- [x] **The Reward Ledger:** Implementation of the **Weekly Corral** to track XP independently of quest status.
- [x] **Portable Schema:** Integrated `go:embed` for idempotent database initialization and easier deployment.

#### **Phase 3: Executive Function Hardening (IN PROGRESS)**
- [ ] **Repeating Quest Logic:** Implementing ledger-based checks to reset Daily and Repeating tasks without duplicating records.
- [ ] **The "Bad Brain Day" Momentum Filter:** A UI-triggered backend toggle that restricts the JSON payload to only `is_non_negotiable = 1` tasks.
- [ ] **The Gear Check Lock:** Introducing a pre-flight validation step where quests remain locked until all required `gear_checks` are toggled.

#### **Phase 4: The Ingestion Bridge & Telemetry**
- [ ] **Automated Seeding:** Building a JSON bulk-importer for rapid task generation.
- [ ] **Atmospheric Trigger:** Integrating with `pressure-sentinel` to automatically suggest "Momentum Mode" based on barometric trends.
- [ ] **API Exposure:** Finalizing the headless endpoint for Obsidian Dataview visualization.