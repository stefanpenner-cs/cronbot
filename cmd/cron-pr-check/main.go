// Command cron-pr-check validates a registry change in a PR, diffs it against
// the base ref, builds a markdown comment, and writes it to --output.
//
// The YAML just calls this one CLI, then posts the comment with `gh`.
//
// Usage:
//
//	cron-pr-check --registry registry.json --base-ref origin/main --output comment.md
//
// Exit codes: 0 = valid, 1 = invalid, 2 = usage/IO error.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"

	"cronbot/internal/crondiff"
	"cronbot/internal/prcomment"
	"cronbot/internal/registry"
)

func main() {
	registryPath := flag.String("registry", "", "path to the new registry.json (required)")
	baseRef := flag.String("base-ref", "", "git ref of the PR base (e.g. origin/main)")
	output := flag.String("output", "", "write the comment markdown to this path (required)")
	repoRoot := flag.String("repo-root", ".", "git repo root for git show")
	flag.Parse()

	if *registryPath == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "--registry and --output are required")
		os.Exit(2)
	}

	// 1. Validate the new registry.
	reg, err := registry.Load(*registryPath)
	if err != nil {
		writeOutput(*output, prcomment.PRCheck(false, "", fmt.Sprintf("load %s: %v", *registryPath, err)))
		os.Exit(1)
	}

	errs := reg.Validate()
	valid := len(errs) == 0

	var validationErrors string
	if !valid {
		var b bytes.Buffer
		for _, e := range errs {
			fmt.Fprintln(&b, "  -", e)
		}
		validationErrors = b.String()
	}

	// 2. Diff old vs new registry.
	var oldEntries []registry.Entry
	if *baseRef != "" {
		oldJSON, err := gitShow(*repoRoot, *baseRef, *registryPath)
		if err == nil && len(oldJSON) > 0 {
			oldReg, err := registry.LoadFromBytes(oldJSON)
			if err == nil {
				oldEntries = oldReg.All()
			}
		}
	}

	changes := crondiff.Diff(oldEntries, reg.All())
	planMd := crondiff.PlanMarkdown(changes)

	// 3. Build the comment.
	comment := prcomment.PRCheck(valid, planMd, validationErrors)
	writeOutput(*output, comment)

	if !valid {
		os.Exit(1)
	}
}

func gitShow(dir, ref, path string) ([]byte, error) {
	cmd := exec.Command("git", "show", ref+":"+path)
	cmd.Dir = dir
	return cmd.Output()
}

func writeOutput(path, content string) {
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write output:", err)
		os.Exit(2)
	}
}