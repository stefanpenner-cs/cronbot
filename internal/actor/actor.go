// Package actor classifies a scheduled-run actor login by account durability —
// the signal that decides whether a cron needs re-homing.
//
//	bot / service  -> durable automation; leave it.
//	deprovisioned  -> EMU-anonymized (former employee); runs have STOPPED.
//	human          -> a personal "*_EMU" handle; dies on departure.
//	external       -> a live non-EMU account (mirrored upstream); unowned.
//	none           -> no scheduled run on record.
package actor

import (
	"regexp"
	"strings"
	"sync"
)

// emuSuffix is the GitHub Enterprise Managed User handle suffix (e.g. "_acme").
// Personal EMU logins end with it; set it for your enterprise via SetEMUSuffix.
var (
	mu        sync.RWMutex
	emuSuffix = "_EMU"
	anonRE    = compileAnon(emuSuffix)
)

func compileAnon(suffix string) *regexp.Regexp {
	return regexp.MustCompile(`^[0-9a-f]{20,}` + regexp.QuoteMeta(suffix) + `$`)
}

// SetEMUSuffix overrides the enterprise EMU handle suffix used to tell a personal
// (human) account from an external one.
func SetEMUSuffix(suffix string) {
	mu.Lock()
	defer mu.Unlock()
	emuSuffix = suffix
	anonRE = compileAnon(suffix)
}

// EMUSuffix returns the currently configured EMU suffix.
func EMUSuffix() string {
	mu.RLock()
	defer mu.RUnlock()
	return emuSuffix
}

// Non-App bot actors seen firing crons (GitHub's web merge bot).
var bots = map[string]bool{"web-flow": true}

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
	mu.RLock()
	defer mu.RUnlock()
	switch {
	case login == "":
		return "none"
	case IsBot(login):
		return "bot"
	case IsService(login):
		return "service"
	case anonRE.MatchString(login):
		return "deprovisioned"
	case strings.HasSuffix(login, emuSuffix):
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
