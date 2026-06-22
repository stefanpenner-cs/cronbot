// Command cronreconcile de-registers managed crons that no longer exist in the
// repos. cronguard lets anyone DELETE a cron (removing a schedule is safe — there
// is no actor left to mis-attribute), so removals happen freely in the target
// repos. This janitor keeps the central registry honest: it prunes every entry
// whose cron is absent from the current inventory and reports what it dropped.
//
// Usage:
//
//	cronreconcile --registry registry.json --crons crons.json [--dry-run]
//
// Run it on a schedule (as li-cron[bot]) in the central repo. Exits 0 whether or
// not anything was pruned; exits 2 on an I/O error.
package main

import (
	"flag"
	"fmt"
	"os"

	"fixcron/internal/inventory"
	"fixcron/internal/registry"
)

func main() {
	registryPath := flag.String("registry", "registry.json", "central registry JSON to reconcile")
	cronsPath := flag.String("crons", "crons.json", "current cron inventory (the live set)")
	dryRun := flag.Bool("dry-run", false, "report what would be de-registered without writing")
	flag.Parse()

	reg, err := registry.Load(*registryPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load registry:", err)
		os.Exit(2)
	}
	crons, err := inventory.LoadCrons(*cronsPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load crons inventory:", err)
		os.Exit(2)
	}

	live := map[string]bool{}
	for _, c := range crons {
		live[inventory.Key(c.Repo, c.Path)] = true
	}

	pruned := reg.Reconcile(live)
	if len(pruned) == 0 {
		fmt.Fprintln(os.Stdout, "Registry is in sync; nothing to de-register.")
		return
	}

	fmt.Fprintf(os.Stdout, "De-registering %d cron(s) whose workflow no longer schedules them:\n", len(pruned))
	for _, e := range pruned {
		fmt.Fprintf(os.Stdout, "  - %s :: %s  (was '%s', owner %s)\n", e.Repo, e.Path, e.Expr, e.OwnerTeam)
	}

	if *dryRun {
		fmt.Fprintln(os.Stdout, "\n--dry-run: registry not written.")
		return
	}
	if err := reg.Save(*registryPath); err != nil {
		fmt.Fprintln(os.Stderr, "save registry:", err)
		os.Exit(2)
	}
	fmt.Fprintf(os.Stdout, "\nRegistry updated: %s\n", *registryPath)
}
