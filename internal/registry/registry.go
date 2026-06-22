// Package registry is the central cron catalog: the source of truth for every
// managed cron (which repo, which file, the schedule, the owning team, and the
// request it came from). The intake bot upserts entries here; deadman and
// rehome can consume it. Cadence is not stored — the cron expression already
// encodes it (see cronsched.IntervalDays).
//
// Stored as a JSON array to keep the module dependency-free (matching the rest
// of fixcron). One entry per (repo, path).
package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

// Entry is one managed cron.
type Entry struct {
	Repo      string `json:"repo"`
	Path      string `json:"path"`
	Expr      string `json:"cron_expression"`
	OwnerTeam string `json:"owner_team"`
	Request   string `json:"request"`
}

// Registry is the whole catalog.
type Registry struct {
	Entries []Entry
}

// Key identifies an entry.
func Key(repo, path string) string { return repo + "::" + path }

func (e Entry) Key() string { return Key(e.Repo, e.Path) }

// Load reads a registry JSON file. A missing file is an empty registry.
func Load(path string) (*Registry, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Registry{}, nil
	}
	if err != nil {
		return nil, err
	}
	var entries []Entry
	if len(b) > 0 {
		if err := json.Unmarshal(b, &entries); err != nil {
			return nil, err
		}
	}
	return &Registry{Entries: entries}, nil
}

// Validate returns one error per malformed entry. Every field is required and
// the expression must be a 5-field cron.
func (r *Registry) Validate() []error {
	var errs []error
	for _, e := range r.Entries {
		if e.Repo == "" || e.Path == "" || e.OwnerTeam == "" {
			errs = append(errs, fmt.Errorf("%s: missing required field", e.Key()))
		}
		if len(strings.Fields(e.Expr)) != 5 {
			errs = append(errs, fmt.Errorf("%s: cron_expression %q is not 5 fields", e.Key(), e.Expr))
		}
	}
	return errs
}

// Upsert adds e, or replaces the existing entry with the same (repo, path).
// It reports whether a new entry was added (false = updated in place). Entries
// are kept sorted by key for stable diffs.
func (r *Registry) Upsert(e Entry) (added bool) {
	for i := range r.Entries {
		if r.Entries[i].Key() == e.Key() {
			r.Entries[i] = e
			return false
		}
	}
	r.Entries = append(r.Entries, e)
	sort.SliceStable(r.Entries, func(i, j int) bool { return r.Entries[i].Key() < r.Entries[j].Key() })
	return true
}

// Remove drops the entry with the given key. It reports whether one was
// removed. This is the de-register primitive: a cron leaves the catalog here.
func (r *Registry) Remove(key string) (removed bool) {
	for i := range r.Entries {
		if r.Entries[i].Key() == key {
			r.Entries = append(r.Entries[:i], r.Entries[i+1:]...)
			return true
		}
	}
	return false
}

// Reconcile drops every entry whose cron no longer exists in the repos. live is
// the set of currently-present cron keys (repo::path), e.g. built from the cron
// inventory. It returns the pruned entries so the caller can report what was
// de-registered. This is how removals propagate: cronguard lets a human delete a
// cron, and Reconcile then de-registers it so the catalog matches reality.
func (r *Registry) Reconcile(live map[string]bool) []Entry {
	var kept, pruned []Entry
	for _, e := range r.Entries {
		if live[e.Key()] {
			kept = append(kept, e)
		} else {
			pruned = append(pruned, e)
		}
	}
	r.Entries = kept
	return pruned
}

// Save writes the registry as indented JSON.
func (r *Registry) Save(path string) error {
	b, err := json.MarshalIndent(r.Entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}
