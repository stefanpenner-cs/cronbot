// Package cronfile edits the cron schedule block in a GitHub Actions workflow
// file. It operates on file content as a string — no YAML parser needed because
// the cron block has a predictable structure.
package cronfile

import (
	"strings"
)

// Add inserts a cron schedule into workflow content. If a schedule block
// already exists, it appends to it. If not, it adds a new one under 'on:'.
func Add(content, expr string) string {
	lines := strings.Split(content, "\n")

	// Check if a schedule block already exists.
	if hasSchedule(lines) {
		return appendCronToSchedule(lines, expr)
	}

	// No schedule block — add one under 'on:'.
	return addScheduleBlock(lines, expr)
}

// Update replaces the first cron expression in the file with newExpr.
// If oldExpr is empty, it replaces any cron expression found.
func Update(content, oldExpr, newExpr string) string {
	if oldExpr == "" {
		// Replace the first cron: line.
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			if strings.Contains(line, "cron:") {
				lines[i] = replaceCronExpr(line, newExpr)
				return strings.Join(lines, "\n")
			}
		}
		return content
	}

	// Replace the specific old expression.
	return strings.Replace(content, oldExpr, newExpr, 1)
}

// Remove deletes all cron schedule lines from the content. If the schedule
// block ends up empty, it removes the 'schedule:' key too.
func Remove(content string) string {
	lines := strings.Split(content, "\n")
	var out []string
	scheduleIndent := -1

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "schedule:") {
			scheduleIndent = indentOf(line)
			// Check if the schedule block has only cron children (which we'll remove).
			// Keep the schedule: line for now; remove it later if it's orphaned.
			continue
		}

		// Skip lines that are cron entries under schedule.
		if scheduleIndent >= 0 && i > 0 {
			lineIndent := indentOf(line)
			if lineIndent > scheduleIndent && strings.Contains(trimmed, "cron:") {
				continue
			}
		}

		out = append(out, line)
	}

	// Remove orphaned 'schedule:' lines (no children left).
	final := removeOrphanedSchedule(out)

	return strings.Join(final, "\n")
}

// --- helpers ---

func hasSchedule(lines []string) bool {
	for _, line := range lines {
		if strings.Contains(strings.TrimSpace(line), "schedule:") {
			return true
		}
	}
	return false
}

func appendCronToSchedule(lines []string, expr string) string {
	// Find the 'schedule:' line and its indentation.
	for i, line := range lines {
		if strings.Contains(strings.TrimSpace(line), "schedule:") {
			indent := indentOf(line)
			// Find where the cron entries under schedule end.
			// We insert after the last cron entry, or right after schedule: if none.
			insertAt := i + 1
			for j := i + 1; j < len(lines); j++ {
				lineIndent := indentOf(lines[j])
				trimmed := strings.TrimSpace(lines[j])
				if lineIndent > indent && (strings.Contains(trimmed, "cron:") || trimmed == "- cron: ...") {
					insertAt = j + 1
					continue
				}
				break
			}
			cronLine := strings.Repeat(" ", indent+2) + "- cron: '" + expr + "'"
			result := make([]string, 0, len(lines)+1)
			result = append(result, lines[:insertAt]...)
			result = append(result, cronLine)
			result = append(result, lines[insertAt:]...)
			return strings.Join(result, "\n")
		}
	}
	return strings.Join(lines, "\n")
}

func addScheduleBlock(lines []string, expr string) string {
	// Find the 'on:' key and insert a schedule block under it.
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "on:" || strings.HasPrefix(trimmed, "on:") {
			indent := indentOf(line)
			scheduleLines := []string{
				strings.Repeat(" ", indent+2) + "schedule:",
				strings.Repeat(" ", indent+4) + "- cron: '" + expr + "'",
			}
			result := make([]string, 0, len(lines)+2)
			result = append(result, lines[:i+1]...)
			result = append(result, scheduleLines...)
			result = append(result, lines[i+1:]...)
			return strings.Join(result, "\n")
		}
	}

	// No 'on:' found — prepend one.
	return "on:\n  schedule:\n    - cron: '" + expr + "'\n" + strings.Join(lines, "\n")
}

func replaceCronExpr(line, newExpr string) string {
	// Replace the cron expression value in a line like "    - cron: '0 9 * * *'"
	idx := strings.Index(line, "cron:")
	if idx < 0 {
		return line
	}
	prefix := line[:idx]
	rest := line[idx:]
	rest = strings.Replace(rest, "'", "", -1)
	if i := strings.Index(rest, ":"); i >= 0 {
		rest = rest[:i+1] + " '" + newExpr + "'"
	}
	return prefix + rest
}

func removeOrphanedSchedule(lines []string) []string {
	var out []string
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "schedule:") {
			out = append(out, line)
			continue
		}

		// Check if any following line is a child of schedule (more indented).
		indent := indentOf(line)
		hasChild := false
		for j := i + 1; j < len(lines); j++ {
			if strings.TrimSpace(lines[j]) == "" {
				continue
			}
			if indentOf(lines[j]) > indent {
				hasChild = true
				break
			}
			break
		}

		if !hasChild {
			continue // skip orphaned schedule: line
		}
		out = append(out, line)
	}
	return out
}

func indentOf(line string) int {
	count := 0
	for _, c := range line {
		if c == ' ' {
			count++
		} else {
			break
		}
	}
	return count
}