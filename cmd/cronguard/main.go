// Command cronguard is the identity-gate required check. It fails a PR that adds,
// changes, or removes a cron unless the PR author is an allowed actor
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
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"cronbot/internal/cronguard"
	"cronbot/internal/flagutil"
)

func main() {
	actor := flag.String("actor", "", "PR author login (github.event.pull_request.user.login)")
	base := flag.String("base", "", "git ref of the PR base (e.g. origin/main)")
	jsonOut := flag.String("json-out", "", "write violations as JSON to this path")
	var allow flagutil.StringList
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
		base, err := gitShow(*base, path)
		if err != nil {
			fmt.Fprintln(os.Stderr, "git show base:", err)
			os.Exit(2)
		}
		diffs = append(diffs, cronguard.FileDiff{
			Path: path,
			Base: base,
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

// gitShow returns the file content at base:path. It returns ("", nil) when the
// file did not exist at base (a new file). A git error (bad ref, missing repo,
// etc.) is returned as an error so the caller can fail loudly instead of
// silently treating every file as newly added.
func gitShow(base, path string) (string, error) {
	if base == "" {
		return "", nil
	}
	cmd := exec.Command("git", "show", base+":"+path)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := stderr.String()
		if strings.Contains(msg, "does not exist") || strings.Contains(msg, "exists on disk") {
			return "", nil
		}
		return "", fmt.Errorf("git show %s:%s: %w: %s", base, path, err, strings.TrimSpace(msg))
	}
	return string(out), nil
}
