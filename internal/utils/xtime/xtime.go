package xtime

import (
	"os"
	"time"
)

// Use the long enough past time as start time, in case timex.Now() - lastTime equals 0.
var initTime = time.Now().AddDate(-1, -1, -1)

// Duration wraps time.ParseDuration but do panic when parse duration occurs.
func Duration(str string) time.Duration {
	dur, err := time.ParseDuration(str)
	if err != nil {
		panic(err)
	}
	return dur
}

// TS RFC 3339 with seconds
// Deprecated: this function will be moved to internal package, user should not use it anymore.
var TS TimeFormat = "2006-01-02 15:04:05"

// TimeFormat ...
// Deprecated: this function will be moved to internal package, user should not use it anymore.
type TimeFormat string

// Format 格式化
// Deprecated: this function will be moved to internal package, user should not use it anymore.
func (ts TimeFormat) Format(t time.Time) string {
	return t.Format(string(ts))
}

// ParseInLocation parses time with location from env "TZ", if "TZ" hasn't been set then we use UTC by default.
func ParseInLocation(layout, value string) (time.Time, error) {
	loc, err := time.LoadLocation(os.Getenv("TZ"))
	if err != nil {
		return time.Time{}, err
	}
	return time.ParseInLocation(layout, value, loc)
}

// Now returns a relative time duration since initTime, which is not important.
// The caller only needs to care about the relative value.
func Now() time.Duration {
	return time.Since(initTime)
}
