package stdlib

import (
	"testing"
	"time"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/tests/require"
)

func TestTimes(t *testing.T) {
	time1 := time.Date(1982, 9, 28, 19, 21, 44, 999, time.Now().Location())
	time2 := time.Now()
	location, _ := time.LoadLocation("Pacific/Auckland")
	time3 := time.Date(1982, 9, 28, 19, 21, 44, 999, location)

	module(t, "times").call(rta, "sleep", 1).expect(rta, core.Undefined)

	r := module(t, "times").call(rta, "since", time.Now().Add(-time.Hour)).o.(core.Value)
	require.True(t, r.Type == core.VT_INT)
	require.True(t, int64(r.Data) > 3600000000000)

	r = module(t, "times").call(rta, "until", time.Now().Add(time.Hour)).o.(core.Value)
	require.True(t, r.Type == core.VT_INT)
	require.True(t, int64(r.Data) < 3600000000000)

	module(t, "times").call(rta, "parse_duration", "1ns").expect(rta, 1)
	module(t, "times").call(rta, "parse_duration", "1ms").expect(rta, 1000000)
	module(t, "times").call(rta, "parse_duration", "1h").expect(rta, 3600000000000)
	module(t, "times").call(rta, "duration_hours", 1800000000000).expect(rta, 0.5)
	module(t, "times").call(rta, "duration_minutes", 1800000000000).expect(rta, 30.0)
	module(t, "times").call(rta, "duration_nanoseconds", 100).expect(rta, 100)
	module(t, "times").call(rta, "duration_seconds", 1000000).expect(rta, 0.001)
	module(t, "times").call(rta, "duration_string", 1800000000000).expect(rta, "30m0s")

	module(t, "times").call(rta, "month_string", 1).expect(rta, "January")
	module(t, "times").call(rta, "month_string", 12).expect(rta, "December")

	module(t, "times").call(rta, "date", 1982, 9, 28, 19, 21, 44, 999).expect(rta, time1)
	module(t, "times").call(rta, "date", 1982, 9, 28, 19, 21, 44, 999, "Pacific/Auckland").expect(rta, time3)

	r = module(t, "times").call(rta, "now").o.(core.Value)
	rt, _ := r.AsTime(rta)
	nowD := time.Until(rt).Nanoseconds()
	require.True(t, 0 > nowD && nowD > -100000000) // within 100ms

	parsed, _ := time.Parse(time.RFC3339, "1982-09-28T19:21:44+07:00")
	module(t, "times").call(rta, "parse", time.RFC3339, "1982-09-28T19:21:44+07:00").expect(rta, parsed)
	module(t, "times").call(rta, "unix", 1234325, 94493).expect(rta, time.Unix(1234325, 94493))

	module(t, "times").call(rta, "add", time2, 3600000000000).expect(rta, time2.Add(time.Duration(3600000000000)))
	module(t, "times").call(rta, "sub", time2, time2.Add(-time.Hour)).expect(rta, 3600000000000)
	module(t, "times").call(rta, "add_date", time2, 1, 2, 3).expect(rta, time2.AddDate(1, 2, 3))
	module(t, "times").call(rta, "after", time2, time2.Add(time.Hour)).expect(rta, false)
	module(t, "times").call(rta, "after", time2, time2.Add(-time.Hour)).expect(rta, true)
	module(t, "times").call(rta, "before", time2, time2.Add(time.Hour)).expect(rta, true)
	module(t, "times").call(rta, "before", time2, time2.Add(-time.Hour)).expect(rta, false)

	module(t, "times").call(rta, "time_year", time1).expect(rta, time1.Year())
	module(t, "times").call(rta, "time_month", time1).expect(rta, int(time1.Month()))
	module(t, "times").call(rta, "time_day", time1).expect(rta, time1.Day())
	module(t, "times").call(rta, "time_hour", time1).expect(rta, time1.Hour())
	module(t, "times").call(rta, "time_minute", time1).expect(rta, time1.Minute())
	module(t, "times").call(rta, "time_second", time1).expect(rta, time1.Second())
	module(t, "times").call(rta, "time_nanosecond", time1).expect(rta, time1.Nanosecond())
	module(t, "times").call(rta, "time_unix", time1).expect(rta, time1.Unix())
	module(t, "times").call(rta, "time_unix_nano", time1).expect(rta, time1.UnixNano())
	module(t, "times").call(rta, "time_format", time1, time.RFC3339).expect(rta, time1.Format(time.RFC3339))
	module(t, "times").call(rta, "is_zero", time1).expect(rta, false)
	module(t, "times").call(rta, "is_zero", time.Time{}).expect(rta, true)
	module(t, "times").call(rta, "to_local", time1).expect(rta, time1.Local())
	module(t, "times").call(rta, "to_utc", time1).expect(rta, time1.UTC())
	module(t, "times").call(rta, "time_location", time1).expect(rta, time1.Location().String())
	module(t, "times").call(rta, "time_string", time1).expect(rta, time1.String())
	module(t, "times").call(rta, "in_location", time1, location.String()).expect(rta, time1.In(location))
}
