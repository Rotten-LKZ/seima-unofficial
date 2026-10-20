-- Migration number: 0001 	 2026-10-20T11:15:00.000Z

CREATE TABLE IF NOT EXISTS streamers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    room_id TEXT NOT NULL,
    uuid TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    streamer_id INTEGER NOT NULL,
    start_time INTEGER NOT NULL,
    end_time INTEGER,
    FOREIGN KEY (streamer_id) REFERENCES streamers(id)
);
