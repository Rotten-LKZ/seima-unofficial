package main

import (
	"encoding/json/v2"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

const bilibiliUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/156.0.0.0 Safari/537.36"

type RoomInfo struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    *struct {
		RoomID     int    `json:"room_id"`
		UID        int    `json:"uid"`
		LiveStatus int    `json:"live_status"`
		LiveTime   string `json:"live_time"` // "2026-10-21 08:27:37" in CST, or "0000-00-00 00:00:00"
	} `json:"data"`
}

var (
	biliCookieMu sync.Mutex
	biliCookie   string
	biliCookieAt time.Time
)

// bilibiliCookie returns a cached buvid3 cookie, refreshing every 6h.
// Bilibili's 412 anti-bot check requires this cookie.
func bilibiliCookie(client *http.Client) string {
	biliCookieMu.Lock()
	defer biliCookieMu.Unlock()
	if biliCookie != "" && time.Since(biliCookieAt) < 6*time.Hour {
		return biliCookie
	}
	req, _ := http.NewRequest("GET", "https://www.bilibili.com/", nil)
	req.Header.Set("User-Agent", bilibiliUA)
	resp, err := client.Do(req)
	if err != nil {
		return "" // proceed without cookie; may still work sometimes
	}
	defer resp.Body.Close()
	var parts []string
	for _, c := range resp.Cookies() {
		parts = append(parts, c.Name+"="+c.Value)
	}
	biliCookie = strings.Join(parts, "; ")
	biliCookieAt = time.Now()
	return biliCookie
}

func fetchRoomInfo(client *http.Client, roomID string) (*RoomInfo, error) {
	url := "https://api.live.bilibili.com/room/v1/Room/get_info?room_id=" + roomID
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", bilibiliUA)
	req.Header.Set("Referer", "https://live.bilibili.com/")
	if ck := bilibiliCookie(client); ck != "" {
		req.Header.Set("Cookie", ck)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bilibili upstream %d", resp.StatusCode)
	}
	var info RoomInfo
	if err := json.UnmarshalRead(resp.Body, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// cstLocation is Asia/Shanghai (UTC+8), the timezone of bilibili's live_time string.
var cstLocation = time.FixedZone("CST", 8*3600)

// parseLiveTime converts "2026-10-21 08:27:37" (CST) to unix milliseconds (UTC).
// Returns 0 for missing or zero-placeholder values.
func parseLiveTime(s string) int64 {
	if s == "" || s == "0000-00-00 00:00:00" {
		return 0
	}
	t, err := time.ParseInLocation("2006-01-02 15:04:05", s, cstLocation)
	if err != nil {
		return 0
	}
	return t.UnixMilli()
}
