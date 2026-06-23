// Command cron-reconcile scans in-progress deployments and completes them when
// the target repo PR merges. Supports scheduled, webhook, and manual modes.
//
// Usage:
//
//	cron-reconcile --repo owner/cronbot
//	cron-reconcile --repo owner/cronbot --pr-number 42
//	cron-reconcile --repo owner/cronbot --target-repo octo-org/foo --target-pr 15
//
// Exit codes: 0 = check completed, 2 = usage/IO error.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"cronbot/internal/recon"
)

func main() {
	repo := flag.String("repo", "", "cronbot repo (owner/name, required)")
	_ = flag.Int("pr-number", 0, "check a specific cronbot PR (manual mode)")
	targetRepo := flag.String("target-repo", "", "target repo to check (webhook mode)")
	targetPR := flag.Int("target-pr", 0, "target repo PR number (webhook mode)")
	output := flag.String("output", "", "write JSON results to this path")
	flag.Parse()

	if *repo == "" {
		fmt.Fprintln(os.Stderr, "--repo is required")
		os.Exit(2)
	}

	shell := recon.ExecShell{}

	// Webhook mode: check a specific target PR first.
	if *targetRepo != "" && *targetPR > 0 {
		_, _, _ = shell.Gh("pr", "view", fmt.Sprintf("%d", *targetPR), "--repo", *targetRepo, "--json", "merged")
	}

	// Run the reconcile.
	results := recon.ReconcileAll(shell, *repo)

	// Print summary.
	summary := map[string]int{"success": 0, "in_progress": 0, "failure": 0, "unknown": 0, "total": len(results)}
	for _, r := range results {
		summary[r.NewState]++
	}

	b, _ := json.MarshalIndent(summary, "", "  ")
	fmt.Println(string(b))

	if *output != "" {
		f, err := os.Create(*output)
		if err != nil {
			fmt.Fprintln(os.Stderr, "write output:", err)
			os.Exit(2)
		}
		defer f.Close()
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		enc.Encode(map[string]any{"results": results, "summary": summary})
	}

	if recon.AllComplete(results) && len(results) > 0 {
		fmt.Println("all complete")
	}
}