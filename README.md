# 🛡️ Quest Log (The Forge)
A Go-based **Relational Task Engine** and CRUD application running on the **Milford Node**. This project implements a gamified, ADHD-friendly productivity framework designed to bridge the gap between high-level creative goals (Novel Publication) and daily executive function.

## 🧠 Project Overview
Quest Log acts as the "Command Center" for the Milford Node ecosystem. By replacing traditional time-blocking with **Task-Based Urgency** and **Dopamine-Linked Rewards**, this tool provides a structured environment to manage shared household operations and individual professional growth.

### 🚀 Key Features
- **The Logic Trio (Defensive Engineering):**
    - **Ghost Guard:** Server-side validation to prevent malformed tasks and ensure data sanitization.
    - **Hard-Coded Economy:** An authoritative XP system (Ducks, Sheep, Cows) to prevent reward-inflation and maintain consistent effort-tracking.
    - **Priority Shield:** A high-visibility triage layer that floats "Non-Negotiable" quests (Migraine Meds, Feeding) to the top of the stack.
- **Momentum Filter (Bad Brain Day Mode):** A global logic toggle that modifies the backend query to reduce the "Pasture" to only 3 critical tasks, preventing choice paralysis during migraine or high-stress windows.
- **The Gear Check (ADHD Lockdown):** A unique "Blocking" logic where complex quests remain locked until all physical supplies (Gear) are toggled as "Gathered."
- **Dopamine Multiplier:** Real-time XP calculation at the point of completion using the formula: `$EarnedXP = (Difficulty * 10) + CurrentStreak$`.
    

## 🛠️ Tech Stack
- **Language:** Go (Golang) 1.2x
- **Database:** SQLite3 (Relational Schema for MLOps migration)
- **Frontend:** HTML5 | CSS3 Flexbox | Go Templates
- **Infrastructure:** Raspberry Pi 3 Model B | Docker | Portainer

## 🗄️ Database Architecture

|**Entity**|**Purpose**|**Key Logic**|
|---|---|---|
|**Quests**|The Core Unit|Difficulty mapping (Duck/Sheep/Cow)|
|**Categories**|Visual Context|Hex-code mapping for `quest-accent-bar`|
|**Gear Checks**|Blocking Unit|Boolean check to unlock Quest "Complete" status|

## 🌐 System Architecture & Integrations

Quest Log is the "Action Layer" of the **Milford Node** ecosystem, designed for cross-platform synergy:

- **[Pressure-Sentinel](https://github.com/Cfirth725/pressure-sentinel):** (Upcoming) Automated trigger to enable "Momentum Mode" when barometric drops are detected.
- **Obsidian Vault:** Headless integration via JSON API to display the "Weekly Corral" dashboard within daily notes.
- **Trail-Sentinel:** Correlation of completed quests against health telemetry to analyze high-productivity windows. (Upcoming) 

## 🚀 Execution Roadmap

#### **Phase 1: The Core Foundation (Complete)**
- Establish Go-SQLite connectivity with unified `RenderTemplate` logic.
- Implement **The Logic Trio**: Ghost Guard (validation), Hard-Coded Economy (XP constants), and Priority Shield (sorting).
- Design dynamic Category loading with Hex-code visual mapping.

#### **Phase 2: Transition & Reward Engine (In Progress)**
- **Atomic Transactions:** Completing the `handleCompleteQuest` logic to ensure data integrity during status changes.
- **The Reward Engine:** Implementing the **Dopamine Multiplier** to calculate XP at the moment of completion based on current streaks.
- **Multi-User Credit:** Logic to handle household tasks—marking quests as `Completed` globally while routing XP rewards exclusively to the `completer_id`.

#### **Phase 3: Executive Function Hardening**
- **The "Bad Brain Day" Momentum Filter:** A UI-triggered backend toggle that restricts the JSON payload to only `is_non_negotiable = 1` tasks to eliminate choice paralysis.
- **The Gear Check Lock:** Introducing a pre-flight validation step. Quests are "Locked" in the UI and cannot transition from `Pending` to `In Progress` until the API validates that all required `gear_checks` are toggled.
- **Multi-User Permissions:** Expanding the `owner_id` logic to support granular permissions between individual and shared household "Pastures."

#### **Phase 4: The Pasture UI & Weekly Corral**
- **Frontend/Backend Sync:** Developing the "Pasture" view where the backend supplies aggregated weekly stats to drive the visual rendering of the sheep assets.
- **The Weekly Corral:** A Sunday night cron-job or Go background worker that archives completed rows and generates the **Weekly Accomplishment Report** for Obsidian.

#### **Phase 5: The Ingestion Bridge & Telemetry**
- **Automated Seeding:** Building a JSON bulk-importer for rapid task generation.
- **Atmospheric Trigger:** Integrating with `pressure-sentinel` to automatically suggest "Momentum Mode" based on barometric trends.
- **API Exposure:** Finalizing the headless endpoint for Obsidian Dataview visualization.