package core_test

import (
	"testing"
	"time"

	"github.com/jokruger/fin128/civil"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/internal/require"
)

func TestValueMarkImmutableDeep(t *testing.T) {
	t.Run("scalar", func(t *testing.T) {
		v := core.IntValue(5)
		v.MarkImmutableDeep()
		require.True(t, v.Immutable)
	})

	t.Run("array with nested array", func(t *testing.T) {
		inner := core.NewArrayValue([]core.Value{core.IntValue(1), core.IntValue(2)}, false)
		outer := core.NewArrayValue([]core.Value{inner, core.IntValue(3)}, false)
		require.False(t, outer.Immutable)
		require.False(t, inner.Immutable)

		outer.MarkImmutableDeep()

		require.True(t, outer.Immutable)
		elems, ok := outer.AsArray()
		require.True(t, ok)
		require.True(t, elems[0].Immutable, "nested array element should have become immutable")
		require.True(t, elems[1].Immutable, "scalar element should have become immutable")

		nested, ok := elems[0].AsArray()
		require.True(t, ok)
		require.True(t, nested[0].Immutable, "doubly-nested element should have become immutable")
		require.True(t, nested[1].Immutable, "doubly-nested element should have become immutable")
	})

	t.Run("dict with nested record", func(t *testing.T) {
		rec := core.NewRecordValue(map[string]core.Value{"x": core.IntValue(1)}, false)
		d := core.NewDictValue(map[string]core.Value{"r": rec}, false)

		d.MarkImmutableDeep()

		require.True(t, d.Immutable)
		m, ok := d.AsDict()
		require.True(t, ok)
		require.True(t, m["r"].Immutable, "nested record should have become immutable")

		rm, ok := m["r"].AsDict()
		require.True(t, ok)
		require.True(t, rm["x"].Immutable, "record field should have become immutable")
	})

	t.Run("record with nested array", func(t *testing.T) {
		arr := core.NewArrayValue([]core.Value{core.IntValue(1), core.IntValue(2)}, false)
		rec := core.NewRecordValue(map[string]core.Value{"items": arr}, false)

		rec.MarkImmutableDeep()

		require.True(t, rec.Immutable)
		m, ok := rec.AsDict()
		require.True(t, ok)
		require.True(t, m["items"].Immutable, "nested array field should have become immutable")

		elems, ok := m["items"].AsArray()
		require.True(t, ok)
		require.True(t, elems[0].Immutable)
		require.True(t, elems[1].Immutable)
	})

	t.Run("error payload", func(t *testing.T) {
		payload := core.NewArrayValue([]core.Value{core.IntValue(1)}, false)
		e := core.NewErrorValue(payload, core.KindUser, errs.CategoryUser, false)
		require.False(t, payload.Immutable)

		e.MarkImmutableDeep()

		require.True(t, e.Immutable)
		got := (*core.Error)(e.Ptr)
		require.True(t, got.Payload.Immutable, "error payload should have become immutable")
		elems, ok := got.Payload.AsArray()
		require.True(t, ok)
		require.True(t, elems[0].Immutable, "element inside error payload should have become immutable")
	})

	t.Run("does not clone: source alias observes the flip", func(t *testing.T) {
		inner := core.NewArrayValue([]core.Value{core.IntValue(1)}, false)
		outer := core.NewArrayValue([]core.Value{inner}, false)

		outer.MarkImmutableDeep()

		// inner was captured by value above (a Value header, not a copy of the underlying array), but its Ptr
		// aliases the same backing Array as the element now stored inside outer — MarkImmutableDeep flips the
		// Immutable flag on the *slot* holding inner inside outer's Elements, not on inner's own local header, so
		// this only demonstrates no cloning happened to the shared backing array, not that the local `inner`
		// variable's header flips too (it can't: Immutable lives in the Value struct itself, not behind Ptr).
		elems, ok := outer.AsArray()
		require.True(t, ok)
		require.True(t, inner.Ptr == elems[0].Ptr)
	})
}

// TestDateStartOfDaySweep pins d.time_in(z) over every day of a decade in zones whose DST changes fall at or
// next to midnight (Santiago and Havana swallow midnight, Tehran and Kyiv do not): the answer is the FIRST
// instant whose local date is d — it reads d, and one nanosecond earlier does not.
func TestDateStartOfDaySweep(t *testing.T) {
	for _, z := range []string{"America/Santiago", "America/Havana", "Asia/Tehran", "Europe/Kyiv", "UTC"} {
		loc, err := core.LoadZone(z)
		require.NoError(t, err)
		first, _ := civil.New(2020, 1, 1)
		for i := int32(0); i < 3653; i++ {
			d, _ := first.AddDays(i)
			v, err := core.DateValue(d).CallMember(nil, member.Unknown, "time_in", []core.Value{core.NewStringValue(z)})
			require.NoError(t, err)
			got, _ := v.AsTime()
			y, m, dd := got.In(loc).Date()
			require.True(t, y == d.Year() && int(m) == int(d.Month()) && dd == d.Day(), "%s %s: %s reads another day", z, d, got)
			py, pm, pd := got.Add(-time.Nanosecond).In(loc).Date()
			require.False(t, py == d.Year() && int(pm) == int(d.Month()) && pd == d.Day(), "%s %s: %s is not the first instant", z, d, got)
		}
	}
}

// Member tables: the ID path, the name fallback, and what SetValueType refuses. Every id and type used here is in
// the user ranges, so nothing collides with a builtin binding.
const (
	tblRead    = member.FirstUserDefined + 500
	tblWrite   = member.FirstUserDefined + 501
	tblBump    = member.FirstUserDefined + 502 // bound, never tabled
	tblUnbound = member.FirstUserDefined + 503 // never bound
)

func init() {
	member.Define(tblRead, "core_test_read")
	member.Define(tblWrite, "core_test_write_in_place")
	member.Define(tblBump, "core_test_bump")
}

func TestMemberTables(t *testing.T) {
	const tt = uint8(200)
	readFn := func(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
		return core.NewStringValue("table:" + id.String()), nil
	}
	err := core.SetValueType(tt, core.ValueTypeDescr{
		Name: func(core.Value) string { return "tabled" },
		Methods: []core.MethodEntry{
			tblRead:  {Fn: readFn, Pure: true},
			tblWrite: {Fn: readFn},
		},
		Properties: []core.PropertyEntry{
			tblRead: {Get: func(_ core.VM, _ core.Value, id member.ID) (core.Value, error) {
				return core.NewStringValue("prop:" + id.String()), nil
			}},
		},
		// a name switch answers its own names and ends in the lookup fallback, like every builtin name switch
		CallNamedMethod: func(vm core.VM, v core.Value, name string, args []core.Value) (core.Value, error) {
			switch name {
			case "core_test_named", "core_test_bump":
				return core.NewStringValue("named:" + name), nil
			}
			return core.CallMemberByLookup(vm, v, name, args)
		},
		IsNamedMethodPure: func(name string) bool { return name == "core_test_named_pure" },
	})
	require.NoError(t, err)
	v := core.Value{Type: tt}

	call := func(id member.ID, name string) string {
		r, err := v.CallMember(nil, id, name, nil)
		require.NoError(t, err)
		s, _ := r.AsString()
		return s
	}
	require.Equal(t, "table:core_test_read", call(tblRead, "core_test_read"), "id path")
	require.Equal(t, "table:core_test_read", call(member.Unknown, "core_test_read"), "id 0: name switch misses, lookup finds the slot")
	require.Equal(t, "named:core_test_bump", call(tblBump, "core_test_bump"), "a bound but empty slot goes to the name switch")
	require.Equal(t, "named:core_test_named", call(member.Unknown, "core_test_named"), "id 0, name-switch hit")
	_, err = v.CallMember(nil, member.Unknown, "whatever", nil)
	require.Error(t, err, "unknown everywhere: invalid_method after the lookup")

	p, err := v.AccessProperty(nil, member.Unknown, "core_test_read")
	require.NoError(t, err)
	require.Equal(t, `"prop:core_test_read"`, p.String())
	_, err = v.AccessProperty(nil, member.Unknown, "nope")
	require.Error(t, err, "default AccessNamedProperty refuses")
	err = v.AssignProperty(nil, tblRead, "core_test_read", core.IntValue(1))
	require.Error(t, err, "Get without Set is read-only: assignment misses to the default AssignNamedProperty")

	require.True(t, core.MemberIsPure(tt, "core_test_read"))
	require.False(t, core.MemberIsPure(tt, "core_test_write_in_place"))
	require.True(t, core.MemberIsPure(tt, "core_test_named_pure"), "an untabled name asks IsNamedMethodPure")
	require.False(t, core.MemberIsPure(tt, "core_test_bump"))

	require.True(t, core.HasMember(tt, "core_test_read"))
	require.False(t, core.HasMember(tt, "core_test_bump"))
	require.False(t, core.HasMember(tt, "whatever"))
}

func TestMemberTablesValidation(t *testing.T) {
	noop := func(core.VM, core.Value, member.ID, []core.Value) (core.Value, error) { return core.Undefined, nil }
	get := func(core.VM, core.Value, member.ID) (core.Value, error) { return core.Undefined, nil }
	cases := []struct {
		name string
		d    core.ValueTypeDescr
	}{
		{"method at an unbound id", core.ValueTypeDescr{Methods: []core.MethodEntry{tblUnbound: {Fn: noop}}}},
		{"property at an unbound id", core.ValueTypeDescr{Properties: []core.PropertyEntry{tblUnbound: {Get: get}}}},
		{"pure _in_place member", core.ValueTypeDescr{Methods: []core.MethodEntry{tblWrite: {Fn: noop, Pure: true}}}},
		{"Methods longer than member.Max", core.ValueTypeDescr{Methods: make([]core.MethodEntry, int(member.Max)+1)}},
		{"Properties longer than member.Max", core.ValueTypeDescr{Properties: make([]core.PropertyEntry, int(member.Max)+1)}},
	}
	for _, c := range cases {
		err := core.SetValueType(201, c.d)
		require.Error(t, err, c.name)
		e, ok := err.(*errs.Error)
		require.True(t, ok, c.name)
		require.Equal(t, errs.KindHost, e.Kind, c.name)
	}
}
