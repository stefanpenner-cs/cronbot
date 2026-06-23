// Package crondiff diffs two registry snapshots into a structured change set.
package crondiff

import (
	"fmt"
	"sort"
	"strings"

	"cronbot/internal/branchname"
	"cronbot/internal/registry"
)

// Action is the kind of change.
type Action string

const (
	Add    Action = "add"
	Remove Action = "remove"
	Update Action = "update"
)

// Change is one registry diff entry.
type Change struct {
	Action   Action           `json:"action"`
	Repo     string           `json:"repo"`
	Path     string           `json:"path"`
	Branch   string           `json:"branch"`
	OldExpr  string           `json:"old_expr,omitempty"`
	NewExpr  string `json:"new_expr,omitempty"`
	OldEntry *registry.Entry `json:"old_entry,omitempty"`
	NewEntry *registry.Entry `json:"new_entry,omitempty"`
}

// Diff compares old and new registry entry slices and returns the changes.
func Diff(old, new []registry.Entry) []Change {
	oldMap := map[string]registry.Entry{}
	for _, e := range old {
		oldMap[e.Key()] = e
	}
	newMap := map[string]registry.Entry{}
	for _, e := range new {
		newMap[e.Key()] = e
	}

	allKeys := map[string]bool{}
	for k := range oldMap {
		allKeys[k] = true
	}
	for k := range newMap {
		allKeys[k] = true
	}

	var out []Change
	for k := range allKeys {
		oldE, hasOld := oldMap[k]
		newE, hasNew := newMap[k]

		switch {
		case hasNew && !hasOld:
			ne := newE
			out = append(out, Change{
				Action:   Add,
				Repo:     newE.Repo,
				Path:     newE.Path,
				Branch:   branchname.Deploy(newE.Repo, newE.Path),
				NewExpr:  newE.Expr,
				NewEntry: &ne,
			})
		case hasOld && !hasNew:
			oe := oldE
			out = append(out, Change{
				Action:   Remove,
				Repo:     oldE.Repo,
				Path:     oldE.Path,
				Branch:   branchname.Deploy(oldE.Repo, oldE.Path),
				OldExpr:  oldE.Expr,
				OldEntry: &oe,
			})
		case hasOld && hasNew:
			if oldE.Expr != newE.Expr {
				oe, ne := oldE, newE
				out = append(out, Change{
					Action:   Update,
					Repo:     newE.Repo,
					Path:     newE.Path,
					Branch:   branchname.Deploy(newE.Repo, newE.Path),
					OldExpr:  oldE.Expr,
					NewExpr:  newE.Expr,
					OldEntry: &oe,
					NewEntry: &ne,
				})
			}
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Action != out[j].Action {
			return out[i].Action < out[j].Action
		}
		return out[i].Repo+"::"+out[i].Path < out[j].Repo+"::"+out[j].Path
	})

	return out
}

// PlanMarkdown renders a change set as a markdown deploy plan.
func PlanMarkdown(changes []Change) string {
	if len(changes) == 0 {
		return "No registry changes detected."
	}

	var b strings.Builder
	b.WriteString("## 📋 Deploy Plan\n\n")

	var adds, removes, updates []Change
	for _, c := range changes {
		switch c.Action {
		case Add:
			adds = append(adds, c)
		case Remove:
			removes = append(removes, c)
		case Update:
			updates = append(updates, c)
		}
	}

	if len(adds) > 0 {
		b.WriteString("### ➕ Added\n")
		for _, c := range adds {
			b.WriteString(fmt.Sprintf("- `%s` :: `%s` → cron: `%s`\n", c.Repo, c.Path, c.NewExpr))
		}
		b.WriteString("\n")
	}

	if len(removes) > 0 {
		b.WriteString("### 🗑️ Removed\n")
		for _, c := range removes {
			b.WriteString(fmt.Sprintf("- `%s` :: `%s`\n", c.Repo, c.Path))
		}
		b.WriteString("\n")
	}

	if len(updates) > 0 {
		b.WriteString("### ✏️ Changed\n")
		for _, c := range updates {
			b.WriteString(fmt.Sprintf("- `%s` :: `%s` → `%s` → `%s`\n", c.Repo, c.Path, c.OldExpr, c.NewExpr))
		}
		b.WriteString("\n")
	}

	return b.String()
}