package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/yalochat/agentic-sdlc/internal/engine"
	"github.com/yalochat/agentic-sdlc/internal/phase"
	"github.com/yalochat/agentic-sdlc/internal/server"
	"github.com/yalochat/agentic-sdlc/internal/store"
)

// migrateIfNeeded imports a legacy state.json into the DB if the DB is empty.
func migrateIfNeeded(st store.Store) {
	runs, err := st.ListRuns()
	if err != nil || len(runs) > 0 {
		return // DB already has data
	}
	state, err := engine.LoadStateFromFile("state.json")
	if err != nil {
		return // No state.json to migrate
	}
	if err := st.ImportState(state); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to migrate state.json: %v\n", err)
		return
	}
	if err := os.Rename("state.json", "state.json.migrated"); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to rename state.json: %v\n", err)
	} else {
		fmt.Println("Migrated state.json → state.json.migrated")
	}
}

func main() {
	port := 3000
	if v := os.Getenv("PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			port = p
		}
	}

	dbPath := "sdlc.db"
	if v := os.Getenv("DB_PATH"); v != "" {
		dbPath = v
	}

	st, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open database: %v\n", err)
		os.Exit(1)
	}
	migrateIfNeeded(st)

	eng := engine.NewEmpty(st)
	phase.RegisterAll(eng)

	srv := server.New(eng, st, port)
	fmt.Printf("Starting server on http://localhost:%d\n", port)
	if err := srv.Start(":" + strconv.Itoa(port)); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
