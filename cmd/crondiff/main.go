// Command crondiff diffs two registry.json files and outputs either a JSON
// change set or a markdown deploy plan.
//
// Usage:
//
//	crondiff --old old.json --new new.json              # JSON to stdout
//	crondiff --old old.json --new new.json --markdown   # markdown plan to stdout
//
// Exit codes: 0 = diff produced (even if empty), 2 = usage/IO error.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"cronbot/internal/crondiff"
	"cronbot/internal/registry"
)

func main() {
	oldPath := flag.String("old", "", "old registry.json ('' = empty)")
	newPath := flag.String("new", "", "new registry.json ('' = empty)")
	markdown := flag.Bool("markdown", false, "output markdown deploy plan instead of JSON")
	flag.Parse()

	old, err := loadEntries(*oldPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load old:", err)
		os.Exit(2)
	}
	new, err := loadEntries(*newPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load new:", err)
		os.Exit(2)
	}

	changes := crondiff.Diff(old, new)

	if *markdown {
		fmt.Print(crondiff.PlanMarkdown(changes))
		return
	}

	b, err := json.MarshalIndent(changes, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "marshal:", err)
		os.Exit(2)
	}
	fmt.Println(string(b))
}

func loadEntries(path string) ([]registry.Entry, error) {
	if path == "" {
		return nil, nil
	}
	reg, err := registry.Load(path)
	if err != nil {
		return nil, err
	}
	return reg.All(), nil
}