package main

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func readAppointments(dir, roomID string) ([][]string, error) {
	if !filepath.IsLocal(roomID) {
		return nil, fmt.Errorf("invalid room_id")
	}

	data, err := os.ReadFile(filepath.Join(dir, roomID+".json"))
	if errors.Is(err, os.ErrNotExist) {
		return [][]string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read appointments: %w", err)
	}

	var appointments [][]string
	if err := json.Unmarshal(data, &appointments); err != nil {
		return nil, fmt.Errorf("decode appointments: %w", err)
	}
	if len(appointments) != 7 {
		return nil, fmt.Errorf("appointments must contain seven weekday arrays")
	}
	for i := range appointments {
		if appointments[i] == nil {
			appointments[i] = []string{}
		}
	}
	return appointments, nil
}
