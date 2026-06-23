package flagutil

import (
	"flag"
	"strings"
	"testing"
)

func TestStringListCollectsRepeated(t *testing.T) {
	var s StringList
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.Var(&s, "x", "")
	args := []string{"-x", "a", "-x", "b", "-x", "c"}
	if err := fs.Parse(args); err != nil {
		t.Fatal(err)
	}
	if len(s) != 3 || s[0] != "a" || s[2] != "c" {
		t.Fatalf("want [a b c], got %v", []string(s))
	}
	if got := s.String(); !strings.Contains(got, "a") {
		t.Fatalf("String() = %q", got)
	}
}
