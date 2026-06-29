// Copyright (c) the go-ruby-time/time authors
//
// SPDX-License-Identifier: BSD-3-Clause

// Package time is a pure-Go (no cgo) reimplementation of MRI 4.0.5's Ruby
// Time class and the `time` standard library's parsing extensions.
//
// A Time is an instant on the timeline modelled as a whole number of seconds
// since the Unix epoch plus a sub-second nanosecond field, carried alongside a
// UTC offset (in seconds east of UTC) and a flag distinguishing an instant that
// was constructed as UTC (rendered "UTC") from one merely at a +00:00 offset
// (rendered "+0000"). All civil and zone arithmetic is done with Go's standard
// time package; this package only matches MRI's API surface and formatting.
//
// The package is deterministic: its only source of non-determinism — the wall
// clock behind Now — is a package-level seam (Now) a test can pin.
package time

import (
	stdtime "time"
)

// Time is the Ruby Time value: an instant carried as (Unix-seconds, nanoseconds
// in [0,1e9)) together with the UTC offset it is observed at and whether it was
// built as UTC. It is immutable; every transformation returns a new *Time.
type Time struct {
	sec    int64 // whole seconds since the Unix epoch (UTC)
	nsec   int32 // sub-second nanoseconds in [0, 1_000_000_000)
	offset int   // seconds east of UTC for the observed zone
	isUTC  bool  // true when constructed as UTC (renders "UTC", not "+0000")
}

// Now is the seam for the wall clock — the only non-deterministic input to the
// package. NowFunc defaults to the real clock; tests override it to pin "now".
// go-composites/time deliberately omits a clock for the same reason, so this is
// where the host's non-determinism is isolated.
var NowFunc = func() (sec int64, nsec int32) {
	t := stdtime.Now()
	return t.Unix(), int32(t.Nanosecond())
}

// normNsec splits a possibly out-of-range (sec, nsec) into a canonical instant
// whose nsec lies in [0, 1e9), borrowing/carrying whole seconds as needed.
func normNsec(sec int64, nsec int64) (int64, int32) {
	sec += nsec / 1_000_000_000
	nsec %= 1_000_000_000
	if nsec < 0 {
		nsec += 1_000_000_000
		sec--
	}
	return sec, int32(nsec)
}

// Now returns the current instant in the local zone, like Ruby's Time.now. The
// instant comes from the NowFunc seam, so it is deterministic under test.
func Now() *Time {
	sec, nsec := NowFunc()
	return &Time{sec: sec, nsec: nsec, offset: localOffset(sec), isUTC: false}
}

// At builds a Time from a Unix timestamp, like Time.at(sec[, subsec, unit]). The
// instant is observed in the local zone. Fractional seconds in sec are honoured.
func At(sec float64) *Time {
	whole := int64(sec)
	frac := int64((sec - float64(whole)) * 1e9)
	s, n := normNsec(whole, frac)
	return &Time{sec: s, nsec: n, offset: localOffset(s), isUTC: false}
}

// AtNsec builds a Time from whole Unix seconds plus a nanosecond addend, observed
// in the local zone — the Time.at(sec, n, :nanosecond) form without floats.
func AtNsec(sec, nsec int64) *Time {
	s, n := normNsec(sec, nsec)
	return &Time{sec: s, nsec: n, offset: localOffset(s), isUTC: false}
}

// AtTime returns the same instant as t, observed in the local zone, like
// Time.at(time) — the offset/UTC flag are taken from the local zone, not t.
func AtTime(t *Time) *Time {
	return &Time{sec: t.sec, nsec: t.nsec, offset: localOffset(t.sec), isUTC: false}
}

// fromCivil assembles a Time from broken-down civil fields at the given offset.
// When utc is true the instant is flagged UTC (renders "UTC"); month/day/etc are
// normalised by Go's time.Date (so e.g. day 0 underflows to the prior month).
func fromCivil(year, month, day, hour, min, sec, nsec, offset int, utc bool) *Time {
	loc := stdtime.FixedZone("", offset)
	g := stdtime.Date(year, stdtime.Month(month), day, hour, min, sec, nsec, loc)
	return &Time{sec: g.Unix(), nsec: int32(g.Nanosecond()), offset: offset, isUTC: utc}
}

// UTC builds a UTC Time from civil fields, like Time.utc / Time.gm. Trailing
// fields default to the start of their range (month 1, day 1, the rest 0).
func UTC(year int, rest ...int) *Time {
	m, d, h, mi, s, ns := civilDefaults(rest)
	return fromCivil(year, m, d, h, mi, s, ns, 0, true)
}

// Local builds a Time from civil fields in the local zone, like Time.local /
// Time.mktime. It renders with a numeric offset, not "UTC".
func Local(year int, rest ...int) *Time {
	m, d, h, mi, s, ns := civilDefaults(rest)
	// Resolve the local offset for the assembled wall-clock instant.
	loc := stdtime.Local
	g := stdtime.Date(year, stdtime.Month(m), d, h, mi, s, ns, loc)
	_, off := g.Zone()
	return &Time{sec: g.Unix(), nsec: int32(g.Nanosecond()), offset: off, isUTC: false}
}

// New builds a Time from civil fields at an explicit UTC offset (seconds east),
// like Time.new(y,m,d,h,mi,s, "+02:00"). A zero offset still renders numerically
// ("+0000"), distinguishing it from Time.utc.
func New(year, month, day, hour, min, sec, offset int) *Time {
	return fromCivil(year, month, day, hour, min, sec, 0, offset, false)
}

// civilDefaults fills the optional civil fields (month..nsec) from a variadic
// tail, defaulting month/day to 1 and the time-of-day fields to 0.
func civilDefaults(rest []int) (month, day, hour, min, sec, nsec int) {
	month, day = 1, 1
	get := func(i, def int) int {
		if i < len(rest) {
			return rest[i]
		}
		return def
	}
	month = get(0, 1)
	day = get(1, 1)
	hour = get(2, 0)
	min = get(3, 0)
	sec = get(4, 0)
	nsec = get(5, 0)
	return
}

// localOffset returns the local zone's UTC offset (seconds east) at instant sec.
func localOffset(sec int64) int {
	_, off := stdtime.Unix(sec, 0).In(stdtime.Local).Zone()
	return off
}

// goTime materialises the receiver as a Go time.Time at its observed offset, the
// shared bridge to the standard library's civil/zone routines.
func (t *Time) goTime() stdtime.Time {
	loc := stdtime.FixedZone(t.zoneName(), t.offset)
	return stdtime.Unix(t.sec, int64(t.nsec)).In(loc)
}

// zoneName is the abbreviated zone label: "UTC" for a UTC instant, otherwise the
// empty string MRI reports (its #zone is nil for a fixed numeric offset).
func (t *Time) zoneName() string {
	if t.isUTC {
		return "UTC"
	}
	return ""
}
