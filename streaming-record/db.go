package main

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

const schema = `
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
`

func openDB(path string) *sql.DB {
	db, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)")
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	if _, err := db.Exec(schema); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	return db
}
