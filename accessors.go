// Copyright (c) the go-ruby-time/time authors
//
// SPDX-License-Identifier: BSD-3-Clause

package time

import (
	"math/big"
	stdtime "time"
)

// Year is the proleptic Gregorian year, like Time#year.
func (t *Time) Year() int { return t.goTime().Year() }

// Month is the month 1..12, like Time#mon / Time#month.
func (t *Time) Month() int { return int(t.goTime().Month()) }

// Day is the day of the month 1..31, like Time#day / Time#mday.
func (t *Time) Day() int { return t.goTime().Day() }

// Hour is the hour of the day 0..23, like Time#hour.
func (t *Time) Hour() int { return t.goTime().Hour() }

// Min is the minute of the hour 0..59, like Time#min.
func (t *Time) Min() int { return t.goTime().Minute() }

// Sec is the second of the minute 0..60, like Time#sec.
func (t *Time) Sec() int { return t.goTime().Second() }

// USec is the sub-second microseconds, like Time#usec / Time#tv_usec.
func (t *Time) USec() int { return int(t.nsec) / 1000 }

// NSec is the sub-second nanoseconds, like Time#nsec / Time#tv_nsec.
func (t *Time) NSec() int { return int(t.nsec) }

// Subsec is the fractional second as an exact rational, like Time#subsec.
func (t *Time) Subsec() *big.Rat {
	return new(big.Rat).SetFrac64(int64(t.nsec), 1_000_000_000)
}

// WDay is the day of the week 0..6 with Sunday=0, like Time#wday.
func (t *Time) WDay() int { return int(t.goTime().Weekday()) }

// YDay is the day of the year 1..366, like Time#yday.
func (t *Time) YDay() int { return t.goTime().YearDay() }

// UTCOffset is the offset from UTC in seconds east, like Time#utc_offset / #gmtoff.
func (t *Time) UTCOffset() int { return t.offset }

// Zone is the abbreviated zone name ("UTC") or "" for a fixed numeric offset,
// like Time#zone (which is nil — represented here as the empty string).
func (t *Time) Zone() string { return t.zoneName() }

// GMT reports whether the instant is UTC, like Time#gmt? / Time#utc?.
func (t *Time) GMT() bool { return t.isUTC }

// DST reports daylight-saving observance. Fixed-offset instants are never DST,
// matching Time#dst? / #isdst for instants built from offsets.
func (t *Time) DST() bool { return false }

// ToI is the whole Unix seconds, like Time#to_i / Time#tv_sec.
func (t *Time) ToI() int64 { return t.sec }

// ToF is the Unix time as a float seconds.nanoseconds, like Time#to_f.
func (t *Time) ToF() float64 { return float64(t.sec) + float64(t.nsec)/1e9 }

// ToR is the Unix time as an exact rational, like Time#to_r.
func (t *Time) ToR() *big.Rat {
	r := new(big.Rat).SetFrac64(int64(t.nsec), 1_000_000_000)
	return r.Add(r, new(big.Rat).SetInt64(t.sec))
}

// ToA is the broken-down 10-tuple [sec, min, hour, mday, mon, year, wday, yday,
// isdst, zone], like Time#to_a. zone is the empty string for a numeric offset.
func (t *Time) ToA() []any {
	return []any{t.Sec(), t.Min(), t.Hour(), t.Day(), t.Month(), t.Year(),
		t.WDay(), t.YDay(), t.DST(), t.zoneName()}
}

// UTCTime returns the same instant flagged UTC, like Time#getutc / #utc / #gmtime.
func (t *Time) UTCTime() *Time {
	return &Time{sec: t.sec, nsec: t.nsec, offset: 0, isUTC: true}
}

// GetLocal returns the same instant observed at offset (seconds east of UTC),
// like Time#getlocal(utc_offset) / Time#localtime(utc_offset).
func (t *Time) GetLocal(offset int) *Time {
	return &Time{sec: t.sec, nsec: t.nsec, offset: offset, isUTC: false}
}

// Local returns the same instant observed in the process's local zone, like
// Time#getlocal / Time#localtime with no argument.
func (t *Time) ToLocal() *Time {
	return &Time{sec: t.sec, nsec: t.nsec, offset: localOffset(t.sec), isUTC: false}
}

// Weekday predicates, like Time#sunday? … Time#saturday?.
func (t *Time) Sunday() bool    { return t.goTime().Weekday() == stdtime.Sunday }
func (t *Time) Monday() bool    { return t.goTime().Weekday() == stdtime.Monday }
func (t *Time) Tuesday() bool   { return t.goTime().Weekday() == stdtime.Tuesday }
func (t *Time) Wednesday() bool { return t.goTime().Weekday() == stdtime.Wednesday }
func (t *Time) Thursday() bool  { return t.goTime().Weekday() == stdtime.Thursday }
func (t *Time) Friday() bool    { return t.goTime().Weekday() == stdtime.Friday }
func (t *Time) Saturday() bool  { return t.goTime().Weekday() == stdtime.Saturday }
