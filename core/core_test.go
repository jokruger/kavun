package core_test

import (
	"testing"
	"time"

	"github.com/jokruger/fin128/civil"

	"github.com/jokruger/kavun/core"
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
			v, err := core.DateValue(d).MethodCall(nil, "time_in", []core.Value{core.NewStringValue(z)})
			require.NoError(t, err)
			got, _ := v.AsTime()
			y, m, dd := got.In(loc).Date()
			require.True(t, y == d.Year() && int(m) == int(d.Month()) && dd == d.Day(), "%s %s: %s reads another day", z, d, got)
			py, pm, pd := got.Add(-time.Nanosecond).In(loc).Date()
			require.False(t, py == d.Year() && int(pm) == int(d.Month()) && pd == d.Day(), "%s %s: %s is not the first instant", z, d, got)
		}
	}
}
