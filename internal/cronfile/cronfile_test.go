package cronfile

import (
	"strings"
	"testing"
)

const fileWithSchedule = `name: Nightly
on:
  schedule:
    - cron: '0 9 * * *'
  push:
    branches: [main]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - run: echo hello
`

const fileNoSchedule = `name: Nightly
on:
  push:
    branches: [main]
jobs:
  build:
    runs-on: ubuntu-latest
`

const fileNoOn = `name: Nightly
jobs:
  build:
    runs-on: ubuntu-latest
`

const fileEmptyCron = `name: Nightly
on:
  schedule:
    - cron: ''
jobs:
  build:
    runs-on: ubuntu-latest
`

func TestAddToExistingSchedule(t *testing.T) {
	result := Add(fileWithSchedule, "0 10 * * *")
	if !strings.Contains(result, "0 10 * * *") {
		t.Fatalf("new cron not in result:\n%s", result)
	}
	if !strings.Contains(result, "0 9 * * *") {
		t.Fatalf("existing cron should be preserved:\n%s", result)
	}
}

func TestAddToNoSchedule(t *testing.T) {
	result := Add(fileNoSchedule, "0 9 * * *")
	if !strings.Contains(result, "schedule:") {
		t.Fatalf("schedule block not added:\n%s", result)
	}
	if !strings.Contains(result, "0 9 * * *") {
		t.Fatalf("cron not in result:\n%s", result)
	}
	if !strings.Contains(result, "push:") {
		t.Fatalf("push block should survive:\n%s", result)
	}
}

func TestAddToNoOn(t *testing.T) {
	result := Add(fileNoOn, "0 9 * * *")
	if !strings.HasPrefix(result, "on:") {
		t.Fatalf("on: should be prepended:\n%s", result)
	}
	if !strings.Contains(result, "0 9 * * *") {
		t.Fatalf("cron not in result:\n%s", result)
	}
}

func TestUpdateReplacesFirstCron(t *testing.T) {
	result := Update(fileWithSchedule, "0 9 * * *", "0 10 * * *")
	if strings.Contains(result, "0 9 * * *") {
		t.Fatalf("old cron should be gone:\n%s", result)
	}
	if !strings.Contains(result, "0 10 * * *") {
		t.Fatalf("new cron should be present:\n%s", result)
	}
}

func TestUpdateWithEmptyOldExpr(t *testing.T) {
	result := Update(fileWithSchedule, "", "0 11 * * *")
	if strings.Contains(result, "0 9 * * *") {
		t.Fatalf("old cron should be gone:\n%s", result)
	}
	if !strings.Contains(result, "0 11 * * *") {
		t.Fatalf("new cron should be present:\n%s", result)
	}
}

func TestUpdateEmptyCron(t *testing.T) {
	result := Update(fileEmptyCron, "", "0 9 * * *")
	if !strings.Contains(result, "0 9 * * *") {
		t.Fatalf("new cron should be present:\n%s", result)
	}
}

func TestRemoveDeletesCronAndSchedule(t *testing.T) {
	result := Remove(fileWithSchedule)
	if strings.Contains(result, "cron:") {
		t.Fatalf("cron should be gone:\n%s", result)
	}
	if strings.Contains(result, "schedule:") {
		t.Fatalf("orphaned schedule: should be removed:\n%s", result)
	}
	if !strings.Contains(result, "push:") {
		t.Fatalf("push block should survive:\n%s", result)
	}
	if !strings.Contains(result, "echo hello") {
		t.Fatalf("job content should survive:\n%s", result)
	}
}

func TestRemoveOnFileWithNoSchedule(t *testing.T) {
	result := Remove(fileNoSchedule)
	if strings.Contains(result, "schedule:") {
		t.Fatalf("should not add schedule:\n%s", result)
	}
	if !strings.Contains(result, "push:") {
		t.Fatalf("push should survive:\n%s", result)
	}
}

func TestRemoveEmptyFile(t *testing.T) {
	if result := Remove(""); result != "" {
		t.Fatalf("empty file should stay empty, got %q", result)
	}
}