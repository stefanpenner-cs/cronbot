// Command rehome emits a DRY-RUN plan to move fragile crons onto a durable
// account by making a schedule-neutral edit to each cron expression.
//
// It applies nothing — no commits, no pushes, no API writes. Inputs are
// produced by the existing pipeline:
//
//	data/cron/octo-org/crons.json      (scripts/cron_inventory.py)
//	data/cron/octo-org/last_runs.json  (scripts/cron_last_runs.py)
//
// Usage (run from the repo root):
//
//	go run ./cmd/rehome
//	go run ./cmd/rehome --json-out reports/cron/rehome_plan.json
//
// Exit codes: 0 = plan generated, 2 = usage/IO error.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"cronbot/internal/inventory"
	"cronbot/internal/rehome"
)

func main() {
	cronsPath := flag.String("crons", "data/cron/octo-org/crons.json", "crons.json path")
	lastRunsPath := flag.String("last-runs", "data/cron/octo-org/last_runs.json", "last_runs.json path")
	jsonOut := flag.String("json-out", "", "write the plan as JSON to this path")
	flag.Parse()

	crons, err := inventory.LoadCrons(*cronsPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load crons:", err)
		os.Exit(2)
	}
	lastRuns, err := inventory.LoadLastRuns(*lastRunsPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load last_runs:", err)
		os.Exit(2)
	}

	rows := rehome.Plan(crons, lastRuns)
	rehome.Emit(rows, os.Stdout)

	if *jsonOut != "" {
		b, err := json.MarshalIndent(rows, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, "marshal:", err)
			os.Exit(2)
		}
		if err := os.WriteFile(*jsonOut, append(b, '\n'), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "write:", err)
			os.Exit(2)
		}
		fmt.Fprintf(os.Stderr, "\nwrote %d plan rows -> %s\n", len(rows), *jsonOut)
	}
}
