// Command cronguard is the identity-gate required check. It fails a PR that adds
// or changes a cron value unless the PR author is an allowed actor
// (cron-bot[bot]), forcing humans through the intake bot.
//
// It compares each changed workflow file's content at --base (a git ref) with
// the version on disk (the PR head), then applies the gate.
//
// Usage in CI:
//
//	cronguard --actor "${{ github.event.pull_request.user.login }}" \
//	          --base "origin/${{ github.base_ref }}" \
//	          $(git diff --name-only "origin/${{ github.base_ref }}"...HEAD -- '.github/workflows/*.yml')
//
// Exit codes: 0 = clean, 1 = unauthorized cron change, 2 = usage/IO error.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"fixcron/internal/cronguard"
)

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error {
	*s = append(*s, v)
	return nil
}

func main() {
	actor := flag.String("actor", "", "PR author login (github.event.pull_request.user.login)")
	base := flag.String("base", "", "git ref of the PR base (e.g. origin/main)")
	jsonOut := flag.String("json-out", "", "write violations as JSON to this path")
	var allow stringList
	flag.Var(&allow, "allow-actor", "actor allowed to change crons; repeatable (default cron-bot[bot])")
	flag.Parse()

	if *actor == "" {
		fmt.Fprintln(os.Stderr, "--actor is required")
		os.Exit(2)
	}

	var diffs []cronguard.FileDiff
	for _, path := range flag.Args() {
		head, err := os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			fmt.Fprintln(os.Stderr, "read head:", err)
			os.Exit(2)
		}
		diffs = append(diffs, cronguard.FileDiff{
			Path: path,
			Base: gitShow(*base, path),
			Head: string(head),
		})
	}

	v := cronguard.Guard(diffs, *actor, allow)
	n := cronguard.Emit(v, os.Stdout)

	if *jsonOut != "" {
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, "marshal:", err)
			os.Exit(2)
		}
		if err := os.WriteFile(*jsonOut, append(b, '\n'), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "write:", err)
			os.Exit(2)
		}
	}

	if n > 0 {
		os.Exit(1)
	}
}

// gitShow returns the file content at base:path, or "" if it did not exist
// there (a new file) or base is unset.
func gitShow(base, path string) string {
	if base == "" {
		return ""
	}
	out, err := exec.Command("git", "show", base+":"+path).Output()
	if err != nil {
		return ""
	}
	return string(out)
}
