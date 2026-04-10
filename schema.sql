CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    dopamine_streak INTEGER DEFAULT 0
);

CREATE TABLE categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    owner_id INTEGER DEFAULT 0, 
    name TEXT NOT NULL,
    color_hex TEXT,
    is_archived BOOLEAN DEFAULT 0,
    UNIQUE(owner_id, name)
);

CREATE TABLE quests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    owner_id INTEGER DEFAULT 0,
    category_id INTEGER,
    title TEXT NOT NULL,
    difficulty INTEGER CHECK( difficulty IN (1, 2, 3) ),
    base_xp INTEGER CHECK( base_xp IN (10, 25, 50) ),
    is_non_negotiable BOOLEAN DEFAULT 0,
    status TEXT DEFAULT 'Pending',
    quest_type TEXT CHECK( quest_type IN ('One-Time', 'Daily', 'Repeating') ), 
    repeat_interval_days INTEGER DEFAULT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_completed_at DATETIME,
    FOREIGN KEY(category_id) REFERENCES categories(id)
);

CREATE TABLE quest_completions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    quest_id INTEGER,
    completed_by_user_id INTEGER,
    completed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    xp_awarded INTEGER,
    FOREIGN KEY(quest_id) REFERENCES quests(id)
);

CREATE TABLE gear_checks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    quest_id INTEGER,
    item_name TEXT NOT NULL,
    is_gathered BOOLEAN DEFAULT 0,
    FOREIGN KEY(quest_id) REFERENCES quests(id)
);