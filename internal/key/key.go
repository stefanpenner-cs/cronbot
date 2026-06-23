// Package key defines the shared (repo, path) key used to identify a managed
// cron across packages (registry, inventory, deadman, rehome).
package key

// Cron is the canonical key for a workflow file's cron: "repo::path".
func Cron(repo, path string) string { return repo + "::" + path }
