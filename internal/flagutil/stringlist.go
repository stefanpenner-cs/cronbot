// Package flagutil provides reusable flag.Value implementations.
package flagutil

import "strings"

// StringList is a flag.Value that collects repeated string flags into a slice.
type StringList []string

func (s *StringList) String() string { return strings.Join(*s, ",") }
func (s *StringList) Set(v string) error {
	*s = append(*s, v)
	return nil
}
