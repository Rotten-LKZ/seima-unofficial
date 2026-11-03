package main

import (
	"encoding/json/v2"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestQueryReturnsAppointments(t *testing.T) {
	tests := []struct {
		name             string
		appointmentJSON  string
		wantAppointments [][]string
	}{
		{
			name:             "missing file returns empty array",
			wantAppointments: [][]string{},
		},
		{
			name:            "existing file returns weekly schedule",
			appointmentJSON: `[["20:00"],[],["19:30-21:00"],[],[],[],[]]`,
			wantAppointments: [][]string{
				{"20:00"}, {}, {"19:30-21:00"}, {}, {}, {}, {},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			db := openDB(filepath.Join(dir, "streaming.db"))
			defer db.Close()

			if _, err := db.Exec(`INSERT INTO streamers (name, room_id, uuid) VALUES (?, ?, ?)`, "Test", "42", "test-uuid"); err != nil {
				t.Fatal(err)
			}

			appointmentsDir := filepath.Join(dir, "appointments")
			if test.appointmentJSON != "" {
				if err := os.MkdirAll(appointmentsDir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(appointmentsDir, "42.json"), []byte(test.appointmentJSON), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			s := &server{db: db, appointmentsDir: appointmentsDir}
			req := httptest.NewRequest("GET", "/query?room_id=42&startTime=0&endTime=1", nil)
			res := httptest.NewRecorder()
			s.handleQuery(res, req)
			if res.Code != 200 {
				t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
			}

			var response struct {
				Appointments [][]string `json:"appointments"`
			}
			if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(response.Appointments, test.wantAppointments) {
				t.Errorf("appointments = %#v, want %#v", response.Appointments, test.wantAppointments)
			}
		})
	}
}
