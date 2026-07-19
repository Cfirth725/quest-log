-- ====================================================================
-- -- CORE IDENTITY & SCHEMA INITIALIZATION --
-- ====================================================================

CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    dopamine_streak INTEGER DEFAULT 0
);

-- ====================================================================
-- -- TAXONOMY & VISUAL CONTEXT MAPPING --
-- ====================================================================

CREATE TABLE IF NOT EXISTS categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    owner_id INTEGER DEFAULT 0, 
    name TEXT NOT NULL,
    color_hex TEXT,
    is_archived BOOLEAN DEFAULT 0,
    UNIQUE(owner_id, name)
);

-- ====================================================================
-- -- THE CORE TRANSACTIONAL QUEST ENGINE --
-- ====================================================================

CREATE TABLE IF NOT EXISTS quests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    owner_id INTEGER DEFAULT 0,
    category_id INTEGER,
    title TEXT NOT NULL,
    difficulty INTEGER CHECK( difficulty IN (1, 2, 3) ), -- 1: Coin, 2: Money Bag, 3: Crown
    base_xp INTEGER CHECK( base_xp IN (1, 5, 10) ),
    is_non_negotiable BOOLEAN DEFAULT 0,
    status TEXT DEFAULT 'active',
    quest_type TEXT CHECK( quest_type IN ('One-Time', 'Daily', 'Repeating', 'Weekly') ), 
    repeat_interval_days INTEGER DEFAULT NULL,
    reset_day_of_week INTEGER DEFAULT 0, 
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_completed_at DATETIME,
    deleted_at DATETIME,
    FOREIGN KEY(category_id) REFERENCES categories(id)
);

-- ====================================================================
-- -- IMMUTABLE CHRONICLE ANALYTICS LEDGER --
-- ====================================================================

CREATE TABLE IF NOT EXISTS quest_completions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    quest_id INTEGER,
    completed_by_user_id INTEGER,
    completed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    xp_awarded INTEGER,
    FOREIGN KEY(quest_id) REFERENCES quests(id)
);

-- ====================================================================
-- -- ADVANCED MECHANICAL PRE-FLIGHT LOCKS --
-- ====================================================================

CREATE TABLE IF NOT EXISTS gear_checks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    quest_id INTEGER,
    item_name TEXT NOT NULL,
    is_gathered BOOLEAN DEFAULT 0,
    FOREIGN KEY(quest_id) REFERENCES quests(id)
);