package core_test

import (
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/core/member/members"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/internal/require"
	"github.com/jokruger/kavun/internal/text"
)

// TestStringMembersAgreeWithRunes: the members that read a string in place (never decoding the whole receiver)
// must answer exactly what the same member answers on the materialised form of the same text — value, cached
// symbol count, and error kind — on receivers that exercise every decode boundary (valid text, Unicode blanks,
// NUL, stray and truncated octets, encoded surrogates, a literal U+FFFD) and on arguments of every accepted
// class, valid and invalid, plus the refused classes. The runes type is written out independently and is the
// reference implementation here. Text results compare as OCTETS: a string is its octets, so a built string may fuse
// an inserted escape with its neighbours into one symbol ("\xe2\x82" + "\xa9" reads "₩") where runes keeps the
// escapes apart — the same text, read by each type's own rule.
func TestStringMembersAgreeWithRunes(t *testing.T) {
	atoms := []string{"a", "b", "c", " ", "\t", "\x00", " ", " ", "é", "€", "😀", "�",
		"\xff", "\x80", "\xc3", "\xa9", "\xe2\x82", "\xed\xa0\x80", "\xf0\x9f\x98"}
	rng := rand.New(rand.NewSource(7))
	receivers := []string{"", "a", "ab", "abcabc", "  ab  ", " ab ", "ééé", "€a€", "₩", "a₩é"}
	for i := 0; i < 500; i++ {
		var b strings.Builder
		for n := rng.Intn(7); n > 0; n-- {
			b.WriteString(atoms[rng.Intn(len(atoms))])
		}
		receivers = append(receivers, b.String())
	}

	str := func(s string) core.Value { return core.NewStringValue(s) }
	rns := func(s string) core.Value { return core.NewRunesValue(text.DecodeText(s), false) }
	bts := func(s string) core.Value { return core.NewBytesValue([]byte(s), false) }
	// every argument class the text-content reading accepts or refuses, valid and invalid
	pool := []core.Value{
		str(""), str("a"), str("ab"), str("abc"), str("é"), str(" "), str(" "), str("\xff"), str("\xe2\x82"), str("\xc3"), str("\xa9"), str("€"),
		rns("ab"), rns("\xff"), rns("é"), bts("ab"), bts("\xa9"), bts("\xc3\xa9"), bts(""),
		// runes whose escapes spell a valid sequence ("₩", "é"): as text they would read as one symbol
		core.NewRunesValue([]rune{0xDCE2, 0xDC82, 0xDCA9}, false), core.NewRunesValue([]rune{0xDCC3, 0xDCA9}, false),
		core.RuneValue('a'), core.RuneValue(' '), core.RuneValue(0), core.RuneValue(0xDCFF), core.RuneValue(0xDCA9), core.RuneValue(0xDCE2), core.RuneValue(0xDCC3), core.RuneValue('é'), core.RuneValue(0xFFFD),
		core.ByteValue('a'), core.ByteValue(' '), core.ByteValue(0xff),
		core.IntValue(97), core.IntValue(0xDCFF), core.IntValue(-1), core.IntValue(0x110000),
		core.FloatValue(1.5), core.True, core.Undefined,
	}
	widths := []core.Value{core.IntValue(0), core.IntValue(3), core.IntValue(8), core.IntValue(-1), str("x")}

	norm := func(v core.Value) any {
		switch v.Type {
		case value.String:
			return *(*string)(v.Ptr)
		case value.Runes:
			r, _ := v.AsRunes()
			return text.EncodeText(r)
		}
		return v.Interface()
	}
	kind := func(t *testing.T, err error) string {
		var e *errs.Error
		require.True(t, errors.As(err, &e), "not an errs.Error: %v", err)
		return e.Kind
	}
	check := func(t *testing.T, name string, id member.ID, recv string, args []core.Value) {
		t.Helper()
		got, gotErr := str(recv).CallMember(nil, id, name, args)
		want, wantErr := rns(recv).CallMember(nil, id, name, args)
		label := fmt.Sprintf("%q.%s(%v)", recv, name, args)
		if wantErr != nil {
			require.Error(t, gotErr, "%s: runes raised %v, string answered %v", label, wantErr, got)
			require.Equal(t, kind(t, wantErr), kind(t, gotErr), label)
			return
		}
		require.NoError(t, gotErr, label)
		require.True(t, reflect.DeepEqual(norm(want), norm(got)), "%s: want %#v, got %#v", label, norm(want), norm(got))
		if got.Type == value.String {
			require.True(t, uint64(len(text.DecodeText(*(*string)(got.Ptr)))) == got.Data, "%s: cached symbol count %d", label, got.Data)
		}
	}

	cases := []struct {
		name     string
		id       member.ID
		min, max int
		width    bool // the first argument is a pad width
	}{
		{"has_prefix", members.HasPrefix, 0, 3, false},
		{"has_suffix", members.HasSuffix, 0, 3, false},
		{"remove_prefix", members.RemovePrefix, 0, 3, false},
		{"remove_suffix", members.RemoveSuffix, 0, 3, false},
		{"first", members.First, 0, 2, false},
		{"last", members.Last, 0, 2, false},
		{"trim", members.Trim, 0, 3, false},
		{"trim_start", members.TrimStart, 0, 3, false},
		{"trim_end", members.TrimEnd, 0, 3, false},
		{"pad_start", members.PadStart, 0, 3, true},
		{"pad_end", members.PadEnd, 0, 3, true},
		{"contains", members.Contains, 0, 3, false},
		{"count", members.Count, 0, 3, false},
		{"index", members.Index, 0, 3, false},
		{"index_last", members.IndexLast, 0, 3, false},
		{"replace", members.Replace, 1, 3, false},
	}
	// the pads, exhaustively over every fill: an escape fill joins octets with the receiver's edge, and only an
	// exhaustive sweep is sure to meet each such seam
	for _, name := range []string{"pad_start", "pad_end"} {
		var id member.ID = members.PadStart
		if name == "pad_end" {
			id = members.PadEnd
		}
		t.Run(name+"/every_fill", func(t *testing.T) {
			for _, recv := range receivers {
				for _, w := range widths {
					for _, fill := range pool {
						check(t, name, id, recv, []core.Value{w, fill})
					}
				}
			}
		})
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for _, recv := range receivers {
				for n := c.min; n <= c.max; n++ {
					for rep := 0; rep < 4; rep++ {
						args := make([]core.Value, n)
						for i := range args {
							args[i] = pool[rng.Intn(len(pool))]
						}
						if c.width && n > 0 {
							args[0] = widths[rng.Intn(len(widths))]
						}
						check(t, c.name, c.id, recv, args)
					}
				}
			}
		})
	}
}

// TestStringMembersDoNotDecodeReceiver pins the performance contract of the in-place string members: no
// allocation proportional to the receiver. What remains is the result — one *string header behind Value.Ptr for a
// substring, the run set for a match member — never a decoded copy of the receiver. A row whose member still
// decodes names the task that converts it and its count today (plain build), and is skipped until then.
func TestStringMembersDoNotDecodeReceiver(t *testing.T) {
	s := core.NewStringValue("abcdef")
	padded := core.NewStringValue("  abcdef  ")
	ab := []core.Value{core.NewStringValue("ab")}
	cd := []core.Value{core.NewStringValue("cd")}
	zz := []core.Value{core.NewStringValue("zz")}
	cases := []struct {
		name   string
		id     member.ID
		recv   core.Value
		args   []core.Value
		allocs float64
		task   int     // the plan task that converts the member; 0 once it reads in place
		today  float64 // the count before that task
	}{
		{"has_prefix", members.HasPrefix, s, ab, 0, 0, 0},
		{"has_suffix", members.HasSuffix, s, ab, 0, 0, 0},
		{"remove_prefix hit", members.RemovePrefix, s, ab, 1, 0, 0},
		{"remove_prefix miss", members.RemovePrefix, s, zz, 0, 0, 0},
		{"remove_suffix miss", members.RemoveSuffix, s, zz, 0, 0, 0},
		{"first", members.First, s, nil, 0, 0, 0},
		{"last", members.Last, s, nil, 0, 0, 0},
		{"trim no-op", members.Trim, s, nil, 0, 0, 0},
		{"trim", members.Trim, padded, nil, 1, 0, 0},
		{"trim_start", members.TrimStart, padded, nil, 1, 0, 0},
		{"trim_end", members.TrimEnd, padded, nil, 1, 0, 0},
		{"pad_start no-op", members.PadStart, s, []core.Value{core.IntValue(3)}, 0, 0, 0},
		{"pad_start", members.PadStart, s, []core.Value{core.IntValue(10)}, 2, 0, 0},
		{"pad_end", members.PadEnd, s, []core.Value{core.IntValue(10)}, 2, 0, 0},
		{"contains", members.Contains, s, cd, 1, 0, 0},
		{"count", members.Count, s, cd, 1, 0, 0},
		{"index", members.Index, s, cd, 0, 11, 2},
		{"index_last", members.IndexLast, s, cd, 0, 11, 2},
		{"replace miss", members.Replace, s, []core.Value{core.NewStringValue("zz"), core.NewStringValue("x")}, 1, 12, 6},
		{"replace", members.Replace, s, []core.Value{core.NewStringValue("cd"), core.NewStringValue("x")}, 2, 12, 6},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.task != 0 {
				t.Skipf("turns green in Task %d; today %v allocs", c.task, c.today)
			}
			got := testing.AllocsPerRun(200, func() {
				if _, err := c.recv.CallMember(nil, c.id, c.name, c.args); err != nil {
					t.Fatal(err)
				}
			})
			require.Equal(t, c.allocs, got)
		})
	}
}
