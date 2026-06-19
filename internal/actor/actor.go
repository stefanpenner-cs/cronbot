// Package actor classifies a scheduled-run actor login by account durability —
// the signal that decides whether a cron needs re-homing.
//
//	bot / service  -> durable automation; leave it.
//	deprovisioned  -> EMU-anonymized (former employee); runs have STOPPED.
//	human          -> a personal "*_LinkedIn" handle; dies on departure.
//	external       -> a live non-LinkedIn account (mirrored upstream); unowned.
//	none           -> no scheduled run on record.
//
// Mirrors scripts/cron_owner_burndown.py so the two stay consistent.
package actor

import (
	"regexp"
	"strings"
)

// An EMU handle anonymizes to a long hex hash + "_LinkedIn" on deprovision.
var anonRE = regexp.MustCompile(`^[0-9a-f]{20,}_LinkedIn$`)

// Non-App bot actors seen firing crons.
var bots = map[string]bool{"li-auto-merge": true, "web-flow": true}

// ClassOrder ranks classes most-fragile first (handy for risk sorting).
var ClassOrder = map[string]int{
	"deprovisioned": 0, "none": 1, "external": 2,
	"human": 3, "service": 4, "bot": 5,
}

// classes whose crons should be moved onto a durable account.
var needsRehome = map[string]bool{
	"deprovisioned": true, "human": true, "external": true,
}

var disposition = map[string]string{
	"bot":           "leave (bot-owned)",
	"service":       "leave (svc bot-owned)",
	"deprovisioned": "URGENT re-home (actor deprovisioned)",
	"human":         "re-home (personal account)",
	"external":      "re-home (external account)",
	"none":          "inert (never fired here)",
}

// IsBot reports whether a login is a GitHub App / known bot actor.
func IsBot(login string) bool {
	return login != "" && (strings.HasSuffix(login, "[bot]") || bots[login])
}

// IsService reports whether a login is a svc-* automation account.
func IsService(login string) bool {
	return login != "" && (strings.HasPrefix(login, "svc-") || strings.HasPrefix(login, "svc_"))
}

// Class buckets a scheduled-run actor login by account durability.
func Class(login string) string {
	switch {
	case login == "":
		return "none"
	case IsBot(login):
		return "bot"
	case IsService(login):
		return "service"
	case anonRE.MatchString(login):
		return "deprovisioned"
	case strings.HasSuffix(login, "_LinkedIn"):
		return "human"
	default:
		return "external"
	}
}

// NeedsRehome reports whether a cron fired by an actor of this class should be
// re-homed onto a durable account.
func NeedsRehome(class string) bool { return needsRehome[class] }

// Disposition returns the recommended action text for a cron given its
// run-actor class.
func Disposition(class string) string { return disposition[class] }
