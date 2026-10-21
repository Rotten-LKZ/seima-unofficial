package main

import (
	"database/sql"
	"log"
	"net/http"
	"sync"
	"time"
)

// pollOnce checks every streamer and opens/closes sessions accordingly.
// All times stored as unix milliseconds (UTC).
func pollOnce(db *sql.DB, client *http.Client) {
	rows, err := db.Query("SELECT id, room_id FROM streamers")
	if err != nil {
		log.Printf("poll: list streamers: %v", err)
		return
	}
	type streamer struct {
		id     int64
		roomID string
	}
	var streamers []streamer
	for rows.Next() {
		var s streamer
		if err := rows.Scan(&s.id, &s.roomID); err != nil {
			log.Printf("poll: scan: %v", err)
			rows.Close()
			return
		}
		streamers = append(streamers, s)
	}
	if err := rows.Err(); err != nil {
		log.Printf("poll: iterate streamers: %v", err)
		rows.Close()
		return
	}
	rows.Close()

	var wg sync.WaitGroup
	for _, s := range streamers {
		wg.Add(1)
		go func(s streamer) {
			defer wg.Done()
			if err := pollStreamer(db, client, s.id, s.roomID, time.Now()); err != nil {
				log.Printf("poll room %s: %v", s.roomID, err)
			}
		}(s)
	}
	wg.Wait()
}

func pollStreamer(db *sql.DB, client *http.Client, streamerID int64, roomID string, now time.Time) error {
	info, err := fetchRoomInfo(client, roomID)
	if err != nil {
		return err
	}
	live := info.Code == 0 && info.Data != nil && info.Data.LiveStatus == 1

	var openID int64
	err = db.QueryRow(
		"SELECT id FROM sessions WHERE streamer_id = ? AND end_time IS NULL ORDER BY start_time DESC LIMIT 1",
		streamerID,
	).Scan(&openID)
	hasOpen := err == nil
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	switch {
	case live && !hasOpen:
		startMs := parseLiveTime(info.Data.LiveTime)
		if startMs == 0 {
			startMs = now.UnixMilli()
		}
		_, err := db.Exec(
			"INSERT INTO sessions (streamer_id, start_time, end_time) VALUES (?, ?, NULL)",
			streamerID, startMs,
		)
		if err != nil {
			return err
		}
		log.Printf("room %s: opened session at %d", roomID, startMs)
	case !live && hasOpen:
		endMs := now.UnixMilli()
		_, err := db.Exec("UPDATE sessions SET end_time = ? WHERE id = ?", endMs, openID)
		if err != nil {
			return err
		}
		log.Printf("room %s: closed session %d at %d", roomID, openID, endMs)
	}
	return nil
}

func startPoller(db *sql.DB, client *http.Client) {
	ticker := time.NewTicker(time.Minute)
	go func() {
		pollOnce(db, client) // immediate first run
		for range ticker.C {
			pollOnce(db, client)
		}
	}()
}
