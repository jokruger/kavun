package member_test

import (
	"testing"

	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/core/member/members"
)

func mustPanic(t *testing.T, name string, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s: expected a panic", name)
		}
	}()
	f()
}

func TestDefineLookupString(t *testing.T) {
	id := member.FirstUserDefined + 1
	member.Define(id, "member_test_round_trip")

	if got := member.Lookup("member_test_round_trip"); got != id {
		t.Errorf("Lookup: got %d, want %d", got, id)
	}
	if got := id.String(); got != "member_test_round_trip" {
		t.Errorf("String: got %q", got)
	}
}

func TestUnbound(t *testing.T) {
	if got := member.Lookup("member_test_never_bound"); got != member.Unknown {
		t.Errorf("Lookup of an unbound name: got %d, want Unknown", got)
	}
	if got := member.Unknown.String(); got != "" {
		t.Errorf("Unknown.String(): got %q", got)
	}
	if got := (member.Max - 1).String(); got != "" {
		t.Errorf("String of an unbound id: got %q", got)
	}
	if got := member.Max.String(); got != "" {
		t.Errorf("Max.String(): got %q", got)
	}
}

func TestDefineRefusals(t *testing.T) {
	id := member.FirstUserDefined + 2
	member.Define(id, "member_test_taken")

	mustPanic(t, "Unknown", func() { member.Define(member.Unknown, "member_test_zero") })
	mustPanic(t, "builtin range", func() { member.Define(member.FirstUserDefined-1, "member_test_builtin_range") })
	mustPanic(t, "Max", func() { member.Define(member.Max, "member_test_max") })
	mustPanic(t, "empty name", func() { member.Define(member.FirstUserDefined+3, "") })
	mustPanic(t, "id bound twice", func() { member.Define(id, "member_test_other") })
	mustPanic(t, "name bound twice", func() { member.Define(member.FirstUserDefined+4, "member_test_taken") })
	mustPanic(t, "builtin name", func() { member.Define(member.FirstUserDefined+5, "len") })

	// a refused Define leaves the registry untouched
	if got := (member.FirstUserDefined + 4).String(); got != "" {
		t.Errorf("refused Define bound its id to %q", got)
	}
}

// The builtin constants are bound at init: each constant's String is its name and Lookup gives the constant back.
func TestBuiltinBinding(t *testing.T) {
	cases := []struct {
		id   member.ID
		name string
	}{
		{members.IsTrue, "is_true"}, {members.Len, "len"}, {members.Repeat, "repeat"}, {members.ID, "id"},
		{members.UUID, "uuid"}, {members.IsNaN, "is_nan"}, {members.Max, "max"}, {members.String, "string"},
		{members.HTMLEscape, "html_escape"}, {members.ISOWeek, "iso_week"},
	}
	for _, c := range cases {
		if got := c.id.String(); got != c.name {
			t.Errorf("%d.String(): got %q, want %q", c.id, got, c.name)
		}
		if got := member.Lookup(c.name); got != c.id {
			t.Errorf("Lookup(%q): got %d, want %d", c.name, got, c.id)
		}
	}

	// every builtin id is below the embedder range, and bound ids round-trip
	bound := 0
	for id := member.ID(1); id < member.FirstUserDefined; id++ {
		if name := id.String(); name != "" {
			bound++
			if member.Lookup(name) != id {
				t.Errorf("id %d (%q) does not round-trip", id, name)
			}
		}
	}
	if bound == 0 {
		t.Fatal("no builtin names are bound")
	}
}
