package main

import (
	"database/sql"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"net/http"
	"strconv"
	"strings"
)

type server struct {
	db       *sql.DB
	client   *http.Client
	adminTok string
}

func (s *server) routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /getInfo", s.withCORS(s.handleGetInfo))
	mux.HandleFunc("GET /query", s.withCORS(s.handleQuery))
	mux.HandleFunc("GET /admin/streamers", s.withCORS(s.requireAdmin(s.handleStreamerList)))
	mux.HandleFunc("POST /admin/streamers", s.withCORS(s.requireAdmin(s.handleStreamerCreate)))
	mux.HandleFunc("PUT /admin/streamers/{id}", s.withCORS(s.requireAdmin(s.handleStreamerUpdate)))
	mux.HandleFunc("DELETE /admin/streamers/{id}", s.withCORS(s.requireAdmin(s.handleStreamerDelete)))
	mux.HandleFunc("GET /admin/sessions", s.withCORS(s.requireAdmin(s.handleSessionList)))
	mux.HandleFunc("POST /admin/sessions", s.withCORS(s.requireAdmin(s.handleSessionCreate)))
	mux.HandleFunc("PUT /admin/sessions/{id}", s.withCORS(s.requireAdmin(s.handleSessionUpdate)))
	mux.HandleFunc("DELETE /admin/sessions/{id}", s.withCORS(s.requireAdmin(s.handleSessionDelete)))
	// CORS preflight
	mux.HandleFunc("OPTIONS /", s.withCORS(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	return mux
}

// ---- middleware ----

func allowedOrigin(origin string) bool {
	if origin == "" {
		return false
	}
	// strip scheme
	host := origin
	if i := strings.Index(host, "://"); i >= 0 {
		host = host[i+3:]
	}
	// strip port
	if i := strings.Index(host, ":"); i >= 0 {
		host = host[:i]
	}
	host = strings.TrimSuffix(host, "/")
	return host == "loli.cn.com" || strings.HasSuffix(host, ".loli.cn.com")
}

func (s *server) withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "86400")
		}
		if r.Method == http.MethodOptions {
			if !allowedOrigin(origin) {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func (s *server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tok := ""
		if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
			tok = strings.TrimPrefix(h, "Bearer ")
		} else {
			tok = r.URL.Query().Get("token")
		}
		if s.adminTok == "" || tok != s.adminTok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		next(w, r)
	}
}

// ---- helpers ----

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.MarshalWrite(w, v)
}

func decodeBody(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.UnmarshalRead(r.Body, v)
}

func pathID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

// ---- public handlers ----

func (s *server) handleGetInfo(w http.ResponseWriter, r *http.Request) {
	roomID := r.URL.Query().Get("room_id")
	if roomID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "room_id required"})
		return
	}
	info, err := fetchRoomInfo(s.client, roomID)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, info)
}

type sessionRow struct {
	ID         int64  `json:"id"`
	StreamerID int64  `json:"streamer_id"`
	StartTime  int64  `json:"start_time"`
	EndTime    *int64 `json:"end_time"`
}

func (s *server) handleQuery(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	roomID := q.Get("room_id")
	startSec, err1 := strconv.ParseInt(q.Get("startTime"), 10, 64)
	endSec, err2 := strconv.ParseInt(q.Get("endTime"), 10, 64)
	if roomID == "" || err1 != nil || err2 != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "room_id, startTime, endTime required"})
		return
	}
	if endSec < startSec {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "endTime must be >= startTime"})
		return
	}
	startMs, endMs := startSec*1000, endSec*1000

	rows, err := s.db.Query(`
		SELECT s.id, s.streamer_id, s.start_time, s.end_time
		FROM sessions s
		JOIN streamers st ON st.id = s.streamer_id
		WHERE st.room_id = ?
		  AND s.start_time <= ?
		  AND (s.end_time IS NULL OR s.end_time >= ?)
		ORDER BY s.start_time ASC`,
		roomID, endMs, startMs,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	sessions := []sessionRow{}
	for rows.Next() {
		var sr sessionRow
		if err := rows.Scan(&sr.ID, &sr.StreamerID, &sr.StartTime, &sr.EndTime); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		sessions = append(sessions, sr)
	}
	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"room_id": roomID, "sessions": sessions})
}

// ---- admin: streamers ----

type streamerRow struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	RoomID string `json:"room_id"`
	UUID   string `json:"uuid"`
}

func (s *server) handleStreamerList(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query("SELECT id, name, room_id, uuid FROM streamers ORDER BY id")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()
	list := []streamerRow{}
	for rows.Next() {
		var sr streamerRow
		if err := rows.Scan(&sr.ID, &sr.Name, &sr.RoomID, &sr.UUID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		list = append(list, sr)
	}
	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"streamers": list})
}

func (s *server) handleStreamerCreate(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name   string `json:"name"`
		RoomID string `json:"room_id"`
		UUID   string `json:"uuid"`
	}
	if err := decodeBody(r, &in); err != nil || in.Name == "" || in.RoomID == "" || in.UUID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name, room_id, uuid required"})
		return
	}
	res, err := s.db.Exec("INSERT INTO streamers (name, room_id, uuid) VALUES (?, ?, ?)",
		in.Name, in.RoomID, in.UUID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	id, _ := res.LastInsertId()
	writeJSON(w, http.StatusOK, streamerRow{ID: id, Name: in.Name, RoomID: in.RoomID, UUID: in.UUID})
}

func (s *server) handleStreamerUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var in struct {
		Name   *string `json:"name"`
		RoomID *string `json:"room_id"`
		UUID   *string `json:"uuid"`
	}
	if err := decodeBody(r, &in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	var sets []string
	var args []any
	if in.Name != nil {
		sets = append(sets, "name = ?")
		args = append(args, *in.Name)
	}
	if in.RoomID != nil {
		sets = append(sets, "room_id = ?")
		args = append(args, *in.RoomID)
	}
	if in.UUID != nil {
		sets = append(sets, "uuid = ?")
		args = append(args, *in.UUID)
	}
	if len(sets) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "nothing to update"})
		return
	}
	args = append(args, id)
	res, err := s.db.Exec("UPDATE streamers SET "+strings.Join(sets, ", ")+" WHERE id = ?", args...)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	var sr streamerRow
	err = s.db.QueryRow("SELECT id, name, room_id, uuid FROM streamers WHERE id = ?", id).
		Scan(&sr.ID, &sr.Name, &sr.RoomID, &sr.UUID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, sr)
}

func (s *server) handleStreamerDelete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	tx, err := s.db.Begin()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer tx.Rollback()
	if _, err := tx.Exec("DELETE FROM sessions WHERE streamer_id = ?", id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	res, err := tx.Exec("DELETE FROM streamers WHERE id = ?", id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

// ---- admin: sessions ----

func (s *server) handleSessionList(w http.ResponseWriter, r *http.Request) {
	var rows *sql.Rows
	var err error
	if sid := r.URL.Query().Get("streamer_id"); sid != "" {
		id, perr := strconv.ParseInt(sid, 10, 64)
		if perr != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid streamer_id"})
			return
		}
		rows, err = s.db.Query(
			"SELECT id, streamer_id, start_time, end_time FROM sessions WHERE streamer_id = ? ORDER BY start_time", id)
	} else {
		rows, err = s.db.Query(
			"SELECT id, streamer_id, start_time, end_time FROM sessions ORDER BY start_time")
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()
	list := []sessionRow{}
	for rows.Next() {
		var sr sessionRow
		if err := rows.Scan(&sr.ID, &sr.StreamerID, &sr.StartTime, &sr.EndTime); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		list = append(list, sr)
	}
	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": list})
}

func (s *server) handleSessionCreate(w http.ResponseWriter, r *http.Request) {
	var in struct {
		StreamerID int64  `json:"streamer_id"`
		StartTime  int64  `json:"start_time"`
		EndTime    *int64 `json:"end_time"`
	}
	if err := decodeBody(r, &in); err != nil || in.StreamerID == 0 || in.StartTime == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "streamer_id, start_time required"})
		return
	}
	res, err := s.db.Exec(
		"INSERT INTO sessions (streamer_id, start_time, end_time) VALUES (?, ?, ?)",
		in.StreamerID, in.StartTime, in.EndTime)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	id, _ := res.LastInsertId()
	writeJSON(w, http.StatusOK, sessionRow{ID: id, StreamerID: in.StreamerID, StartTime: in.StartTime, EndTime: in.EndTime})
}

func (s *server) handleSessionUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	// need to distinguish missing vs explicit null for end_time
	var raw map[string]jsontext.Value
	if err := decodeBody(r, &raw); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	var sets []string
	var args []any
	if v, ok := raw["streamer_id"]; ok {
		var x int64
		if err := json.Unmarshal(v, &x); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid streamer_id"})
			return
		}
		sets = append(sets, "streamer_id = ?")
		args = append(args, x)
	}
	if v, ok := raw["start_time"]; ok {
		var x int64
		if err := json.Unmarshal(v, &x); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid start_time"})
			return
		}
		sets = append(sets, "start_time = ?")
		args = append(args, x)
	}
	if v, ok := raw["end_time"]; ok {
		sets = append(sets, "end_time = ?")
		if string(v) == "null" {
			args = append(args, nil)
		} else {
			var x int64
			if err := json.Unmarshal(v, &x); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid end_time"})
				return
			}
			args = append(args, x)
		}
	}
	if len(sets) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "nothing to update"})
		return
	}
	args = append(args, id)
	res, err := s.db.Exec("UPDATE sessions SET "+strings.Join(sets, ", ")+" WHERE id = ?", args...)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	var sr sessionRow
	err = s.db.QueryRow("SELECT id, streamer_id, start_time, end_time FROM sessions WHERE id = ?", id).
		Scan(&sr.ID, &sr.StreamerID, &sr.StartTime, &sr.EndTime)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, sr)
}

func (s *server) handleSessionDelete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	res, err := s.db.Exec("DELETE FROM sessions WHERE id = ?", id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}
