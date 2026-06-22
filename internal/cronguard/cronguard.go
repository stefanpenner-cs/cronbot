// Package cronguard is the identity gate: the required CI check that blocks any
// human-authored cron change unless the change is authored by an allowed actor
// (cron-bot[bot]). Humans are forced through the intake bot for adds, edits, and
// removals alike.
//
// It is diff-aware. It fires on any cron expression that differs between BASE
// and HEAD: added/changed (in HEAD, not BASE) or removed (in BASE, not HEAD).
// Removing a cron also goes through the bot, so the registry never drifts.
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

// Guard returns a violation for every cron expression a non-allowed actor adds
// (present in head, not base) or removes (present in base, not head). The bot
// retires crons through the gated removal flow, so its deletes pass. If allowed
// is empty, DefaultAllowed is used.
func Guard(diffs []FileDiff, actor string, allowed []string) []Violation {
	if len(allowed) == 0 {
		allowed = DefaultAllowed
	}
	if isAllowed(actor, allowed) {
		return nil
	}
	allowList := strings.Join(allowed, ", ")
	var out []Violation
	for _, d := range diffs {
		base := exprSet(d.Base)
		head := exprSet(d.Head)
		var added, removed []string
		for expr := range head {
			if !base[expr] {
				added = append(added, expr)
			}
		}
		for expr := range base {
			if !head[expr] {
				removed = append(removed, expr)
			}
		}
		sort.Strings(added)
		sort.Strings(removed)
		for _, expr := range added {
			out = append(out, Violation{
				Path:  d.Path,
				Expr:  expr,
				Actor: actor,
				Message: fmt.Sprintf(
					"cron added/changed by %q; only %s may change crons — file a cron request instead",
					actor, allowList),
			})
		}
		for _, expr := range removed {
			out = append(out, Violation{
				Path:  d.Path,
				Expr:  expr,
				Actor: actor,
				Message: fmt.Sprintf(
					"cron removed by %q; only %s may remove crons — file a cron-removal request first",
					actor, allowList),
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
