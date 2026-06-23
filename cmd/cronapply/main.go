// Command cronapply edits a workflow file's cron schedule in place.
//
// Usage:
//
//	cronapply --file workflow.yml --action add --expr "0 9 * * *"
//	cronapply --file workflow.yml --action update --old-expr "0 9 * * *" --new-expr "0 10 * * *"
//	cronapply --file workflow.yml --action remove
//	cronapply --file workflow.yml --action remove --dry-run   # print to stdout, don't write
//
// Exit codes: 0 = file edited (or dry-run), 1 = nothing to change, 2 = usage/IO error.
package main

import (
	"flag"
	"fmt"
	"os"

	"cronbot/internal/cronfile"
)

func main() {
	filePath := flag.String("file", "", "workflow YAML file to edit (required)")
	action := flag.String("action", "", "add, update, or remove (required)")
	expr := flag.String("expr", "", "cron expression (for add)")
	oldExpr := flag.String("old-expr", "", "old cron expression (for update)")
	newExpr := flag.String("new-expr", "", "new cron expression (for update)")
	dryRun := flag.Bool("dry-run", false, "print result to stdout, don't write")
	flag.Parse()

	if *filePath == "" || *action == "" {
		fmt.Fprintln(os.Stderr, "--file and --action are required")
		os.Exit(2)
	}

	content, err := os.ReadFile(*filePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read:", err)
		os.Exit(2)
	}

	var result string
	switch *action {
	case "add":
		if *expr == "" {
			fmt.Fprintln(os.Stderr, "--expr is required for add")
			os.Exit(2)
		}
		result = cronfile.Add(string(content), *expr)
	case "update":
		if *newExpr == "" {
			fmt.Fprintln(os.Stderr, "--new-expr is required for update")
			os.Exit(2)
		}
		result = cronfile.Update(string(content), *oldExpr, *newExpr)
	case "remove":
		result = cronfile.Remove(string(content))
	default:
		fmt.Fprintln(os.Stderr, "action must be add, update, or remove")
		os.Exit(2)
	}

	if result == string(content) {
		fmt.Fprintln(os.Stderr, "no changes")
		os.Exit(1)
	}

	if *dryRun {
		fmt.Print(result)
		return
	}

	if err := os.WriteFile(*filePath, []byte(result), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write:", err)
		os.Exit(2)
	}
	fmt.Println("ok")
}