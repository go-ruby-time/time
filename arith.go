// Copyright (c) the go-ruby-time/time authors
//
// SPDX-License-Identifier: BSD-3-Clause

package time

import "math"

// Add returns a new Time shifted forward by secs seconds (which may be
// fractional and negative), like Time#+. The observed offset and UTC flag are
// preserved, matching MRI.
func (t *Time) Add(secs float64) *Time {
	whole := int64(math.Floor(secs))
	frac := int64(math.Round((secs - math.Floor(secs)) * 1e9))
	s, n := normNsec(t.sec+whole, int64(t.nsec)+frac)
	return &Time{sec: s, nsec: n, offset: t.offset, isUTC: t.isUTC}
}

// Sub returns this instant shifted back by secs seconds, like Time#-(numeric).
func (t *Time) Sub(secs float64) *Time { return t.Add(-secs) }

// Diff returns the seconds between two instants as a float, like Time#-(Time)
// which yields a Float (here exact to nanosecond resolution).
func (t *Time) Diff(other *Time) float64 {
	return float64(t.sec-other.sec) + float64(int64(t.nsec)-int64(other.nsec))/1e9
}

// instant returns a comparable (sec, nsec) ordering pair for the receiver.
func (t *Time) instant() (int64, int32) { return t.sec, t.nsec }

// Cmp orders two instants as -1, 0 or 1, like Time#<=> for a Time argument.
func (t *Time) Cmp(other *Time) int {
	as, an := t.instant()
	bs, bn := other.instant()
	switch {
	case as < bs || (as == bs && an < bn):
		return -1
	case as > bs || (as == bs && an > bn):
		return 1
	default:
		return 0
	}
}

// Before reports whether t precedes other, like Time#<.
func (t *Time) Before(other *Time) bool { return t.Cmp(other) < 0 }

// After reports whether t follows other, like Time#>.
func (t *Time) After(other *Time) bool { return t.Cmp(other) > 0 }

// Equal reports whether two instants denote the same point in time, like Time#==
// / Time#eql?. The observed offset is irrelevant — only the instant matters.
func (t *Time) Equal(other *Time) bool { return t.sec == other.sec && t.nsec == other.nsec }

// Hash is a value hash consistent with Equal, like Time#hash: equal instants
// hash equally regardless of their observed offset.
func (t *Time) Hash() uint64 {
	const prime = 1099511628211
	h := uint64(14695981039346656037)
	h = (h ^ uint64(t.sec)) * prime
	h = (h ^ uint64(uint32(t.nsec))) * prime
	return h
}

// roundDiv rounds n/scale to the nearest integer, halves away from zero (the
// behaviour MRI's Time#round uses for the fractional second).
func roundDiv(n, scale int64) int64 {
	if n >= 0 {
		return (n + scale/2) / scale
	}
	return -((-n + scale/2) / scale)
}

// scaleFor returns 10^(9-digits), the nanosecond bucket size for `digits`
// fractional decimal places (clamped to [0,9]).
func scaleFor(digits int) int64 {
	if digits < 0 {
		digits = 0
	}
	if digits > 9 {
		digits = 9
	}
	scale := int64(1)
	for i := 0; i < 9-digits; i++ {
		scale *= 10
	}
	return scale
}

// Round returns the instant with its sub-second part rounded to `digits`
// fractional decimal places, like Time#round(digits).
func (t *Time) Round(digits int) *Time {
	scale := scaleFor(digits)
	buckets := roundDiv(int64(t.nsec), scale)
	s, n := normNsec(t.sec, buckets*scale)
	return &Time{sec: s, nsec: n, offset: t.offset, isUTC: t.isUTC}
}

// Floor returns the instant with its sub-second part truncated towards the past
// to `digits` fractional decimal places, like Time#floor(digits).
func (t *Time) Floor(digits int) *Time {
	scale := scaleFor(digits)
	n := (int64(t.nsec) / scale) * scale
	return &Time{sec: t.sec, nsec: int32(n), offset: t.offset, isUTC: t.isUTC}
}

// Ceil returns the instant with its sub-second part rounded up to `digits`
// fractional decimal places, like Time#ceil(digits).
func (t *Time) Ceil(digits int) *Time {
	scale := scaleFor(digits)
	n := ((int64(t.nsec) + scale - 1) / scale) * scale
	s, nn := normNsec(t.sec, n)
	return &Time{sec: s, nsec: int32(nn), offset: t.offset, isUTC: t.isUTC}
}
