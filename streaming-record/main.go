package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	var (
		addr     = flag.String("addr", ":8787", "listen address")
		dbPath   = flag.String("db", "streaming.db", "sqlite database path")
		adminTok = flag.String("admin-token", os.Getenv("ADMIN_TOKEN"), "admin bearer token (or ADMIN_TOKEN env)")
	)
	flag.Parse()

	if *adminTok == "" {
		log.Println("warning: admin token empty; all /admin/* will return 401")
	}

	db := openDB(*dbPath)
	defer db.Close()

	client := &http.Client{Timeout: 10 * time.Second}

	s := &server{db: db, client: client, adminTok: *adminTok}

	startPoller(db, client)

	log.Printf("listening on %s (db=%s)", *addr, *dbPath)
	log.Fatal(http.ListenAndServe(*addr, s.routes()))
}
