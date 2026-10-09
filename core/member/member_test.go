package member_test

import (
	"testing"

	"github.com/jokruger/kavun/core/member"
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

func TestDefineLookupName(t *testing.T) {
	id := member.FirstUserDefined + 1
	member.Define(id, "member_test_round_trip")

	if got := member.Lookup("member_test_round_trip"); got != id {
		t.Errorf("Lookup: got %d, want %d", got, id)
	}
	if got := member.Name(id); got != "member_test_round_trip" {
		t.Errorf("Name: got %q", got)
	}
	if got := id.String(); got != "member_test_round_trip" {
		t.Errorf("String: got %q", got)
	}
}

func TestUnbound(t *testing.T) {
	if got := member.Lookup("member_test_never_bound"); got != member.Unknown {
		t.Errorf("Lookup of an unbound name: got %d, want Unknown", got)
	}
	if got := member.Name(member.Unknown); got != "" {
		t.Errorf("Name(Unknown): got %q", got)
	}
	if got := member.Name(member.Max - 1); got != "" {
		t.Errorf("Name of an unbound id: got %q", got)
	}
	if got := member.Name(member.Max); got != "" {
		t.Errorf("Name(Max): got %q", got)
	}
}

func TestDefineRefusals(t *testing.T) {
	id := member.FirstUserDefined + 2
	member.Define(id, "member_test_taken")

	mustPanic(t, "Unknown", func() { member.Define(member.Unknown, "member_test_zero") })
	mustPanic(t, "Max", func() { member.Define(member.Max, "member_test_max") })
	mustPanic(t, "empty name", func() { member.Define(member.FirstUserDefined+3, "") })
	mustPanic(t, "id bound twice", func() { member.Define(id, "member_test_other") })
	mustPanic(t, "name bound twice", func() { member.Define(member.FirstUserDefined+4, "member_test_taken") })

	// a refused Define leaves the registry untouched
	if got := member.Name(member.FirstUserDefined + 4); got != "" {
		t.Errorf("refused Define bound its id to %q", got)
	}
}
