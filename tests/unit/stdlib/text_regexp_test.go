package stdlib

import (
	"testing"

	"github.com/jokruger/kavun/core"
)

func TestTextREAlternation(t *testing.T) {
	module(t, "text").call(rta, "re_find", "([a-zA-Z])|([0-9])", "a").expect(rta, ARR{
		ARR{
			IMAP{"text": "a", "begin": 0, "end": 1},
			IMAP{"text": "a", "begin": 0, "end": 1},
		},
	}, "alternation with letter")

	module(t, "text").call(rta, "re_find", "([a-zA-Z])|([0-9])", "5").expect(rta, ARR{
		ARR{
			IMAP{"text": "5", "begin": 0, "end": 1},
			IMAP{"text": "5", "begin": 0, "end": 1},
		},
	}, "alternation with number")

	module(t, "text").call(rta, "re_find", "([a-zA-Z])|([0-9])", "").expect(rta, core.Undefined, "empty input")

	module(t, "text").call(rta, "re_find", "([a-zA-Z])|([0-9])", "!").expect(rta, core.Undefined, "non-matching input")

	module(t, "text").call(rta, "re_find", "(?:([a-zA-Z])|([0-9]))+", "a5b").expect(rta, ARR{
		ARR{
			IMAP{"text": "a5b", "begin": 0, "end": 3},
			IMAP{"text": "b", "begin": 2, "end": 3},
			IMAP{"text": "5", "begin": 1, "end": 2},
		},
	}, "multiple alternations")

	module(t, "text").call(rta, "re_find", "(foo)|(bar)|(baz)", "foo").expect(rta, ARR{
		ARR{
			IMAP{"text": "foo", "begin": 0, "end": 3},
			IMAP{"text": "foo", "begin": 0, "end": 3},
		},
	}, "multiple groups with non-matches")

	module(t, "text").call(rta, "re_find", "((cat)|(dog))((run)|(walk))", "catrun").expect(rta, ARR{
		ARR{
			IMAP{"text": "catrun", "begin": 0, "end": 6},
			IMAP{"text": "cat", "begin": 0, "end": 3},
			IMAP{"text": "cat", "begin": 0, "end": 3},
			IMAP{"text": "run", "begin": 3, "end": 6},
			IMAP{"text": "run", "begin": 3, "end": 6},
		},
	}, "nested groups with alternation")
}
