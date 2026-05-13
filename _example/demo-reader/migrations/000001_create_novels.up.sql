CREATE TABLE IF NOT EXISTS novels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    path TEXT NOT NULL UNIQUE,
    word_count INTEGER DEFAULT 0,
    file_size INTEGER DEFAULT 0,
    scroll_top REAL DEFAULT 0,
    scroll_height REAL DEFAULT 0,
    is_current INTEGER DEFAULT 0,
    created_at TEXT,
    updated_at TEXT
);
