// Command deadman flags GHA crons that have silently stopped.
//
// It compares each cron's expected cadence against the timestamp of its last
// actual `schedule` run and reports the ones that never fired here or have gone
// stale. Inputs are produced by the existing pipeline:
//
//	data/cron/linkedin-actions/crons.json      (scripts/cron_inventory.py)
//	data/cron/linkedin-actions/last_runs.json  (scripts/cron_last_runs.py)
//
// Usage (run from the fix-cron/ module directory):
//
//	go run ./cmd/deadman
//	go run ./cmd/deadman --json-out ../reports/cron/deadman.json
//	go run ./cmd/deadman --all   # include healthy crons in JSON
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"fixcron/internal/deadman"
	"fixcron/internal/inventory"
)

func main() {
	cronsPath := flag.String("crons", "../data/cron/linkedin-actions/crons.json", "crons.json path")
	lastRunsPath := flag.String("last-runs", "../data/cron/linkedin-actions/last_runs.json", "last_runs.json path")
	jsonOut := flag.String("json-out", "", "write the report rows as JSON to this path")
	all := flag.Bool("all", false, "include healthy crons in the JSON output")
	flag.Parse()

	crons, err := inventory.LoadCrons(*cronsPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load crons:", err)
		os.Exit(1)
	}
	lastRuns, err := inventory.LoadLastRuns(*lastRunsPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load last_runs:", err)
		os.Exit(1)
	}

	rows := deadman.Assess(deadman.CollapseFiles(crons), lastRuns, time.Now().UTC())
	flagged := deadman.Missed(rows)
	deadman.Emit(flagged, os.Stdout)

	if *jsonOut != "" {
		payload := flagged
		if *all {
			payload = rows
		}
		b, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, "marshal:", err)
			os.Exit(1)
		}
		if err := os.WriteFile(*jsonOut, b, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "write:", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "\nwrote %d rows -> %s\n", len(payload), *jsonOut)
	}
}
