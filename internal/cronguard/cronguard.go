// Package cronguard is the identity gate: the required CI check that blocks a
// cron value from being added or changed unless the change is authored by an
// allowed actor (cron-bot[bot]). Humans are forced through the intake bot.
//
// It is diff-aware. A cron VALUE change looks like "old value removed, new value
// added", so the gate fires on any cron expression present in HEAD but not in
// BASE. Removing a cron (cleanup) is always allowed.
package cronguard

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"fixcron/internal/cronlint"
)

// DefaultAllowed is the actor permitted to add or change crons.
var DefaultAllowed = []string{"cron-bot[bot]"}

// FileDiff is one workflow file's base and head content. Base is "" for a new
// file; Head is "" for a deleted file.
type FileDiff struct {
	Path string
	Base string
	Head string
}

// Violation is one newly added/changed cron made by a non-allowed actor.
type Violation struct {
	Path    string `json:"path"`
	Expr    string `json:"expr"`
	Actor   string `json:"actor"`
	Message string `json:"message"`
}

func exprSet(content string) map[string]bool {
	out := map[string]bool{}
	for _, c := range cronlint.ParseCrons(content) {
		out[c.Expr] = true
	}
	return out
}

func isAllowed(actor string, allowed []string) bool {
	for _, a := range allowed {
		if strings.EqualFold(a, actor) {
			return true
		}
	}
	return false
}

// Guard returns a violation for every cron expression that appears in a file's
// head but not its base, when actor is not allowed. If allowed is empty,
// DefaultAllowed is used.
func Guard(diffs []FileDiff, actor string, allowed []string) []Violation {
	if len(allowed) == 0 {
		allowed = DefaultAllowed
	}
	if isAllowed(actor, allowed) {
		return nil
	}
	var out []Violation
	for _, d := range diffs {
		base := exprSet(d.Base)
		var added []string
		for expr := range exprSet(d.Head) {
			if !base[expr] {
				added = append(added, expr)
			}
		}
		sort.Strings(added)
		for _, expr := range added {
			out = append(out, Violation{
				Path:  d.Path,
				Expr:  expr,
				Actor: actor,
				Message: fmt.Sprintf(
					"cron added/changed by %q; only %s may merge cron changes — file a cron request instead",
					actor, strings.Join(allowed, ", ")),
			})
		}
	}
	return out
}

// Emit prints a human summary and returns the count.
func Emit(violations []Violation, w io.Writer) int {
	if len(violations) == 0 {
		fmt.Fprintln(w, "No unauthorized cron changes. \u2705")
		return 0
	}
	fmt.Fprintf(w, "%d unauthorized cron change(s):\n\n", len(violations))
	for _, v := range violations {
		fmt.Fprintf(w, "%s  cron: '%s'\n    %s\n", v.Path, v.Expr, v.Message)
	}
	return len(violations)
}
