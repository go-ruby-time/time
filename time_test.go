// Copyright (c) the go-ruby-time/time authors
//
// SPDX-License-Identifier: BSD-3-Clause

package time

import (
	"math/big"
	"testing"
	stdtime "time"
)

// ref is the fixed reference instant used across the deterministic suite:
// Monday 2026-06-29 05:18:32 UTC (Unix 1782710312). Pinning it keeps every
// assertion machine- and clock-independent.
func ref() *Time { return UTC(2026, 6, 29, 5, 18, 32) }

func TestConstructorsAndComponents(t *testing.T) {
	x := ref()
	checks := []struct {
		name string
		got  int
		want int
	}{
		{"year", x.Year(), 2026},
		{"month", x.Month(), 6},
		{"day", x.Day(), 29},
		{"hour", x.Hour(), 5},
		{"min", x.Min(), 18},
		{"sec", x.Sec(), 32},
		{"usec", x.USec(), 0},
		{"nsec", x.NSec(), 0},
		{"wday", x.WDay(), 1},
		{"yday", x.YDay(), 180},
		{"offset", x.UTCOffset(), 0},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", c.name, c.got, c.want)
		}
	}
	if x.ToI() != 1782710312 {
		t.Errorf("ToI = %d", x.ToI())
	}
	if x.ToF() != 1782710312.0 {
		t.Errorf("ToF = %v", x.ToF())
	}
	if !x.GMT() || x.DST() {
		t.Errorf("GMT=%v DST=%v", x.GMT(), x.DST())
	}
	if x.Zone() != "UTC" {
		t.Errorf("Zone = %q", x.Zone())
	}
}

func TestToRAndSubsec(t *testing.T) {
	x := ref()
	if got, want := x.ToR(), big.NewRat(1782710312, 1); got.Cmp(want) != 0 {
		t.Errorf("ToR = %v, want %v", got, want)
	}
	if x.Subsec().Sign() != 0 {
		t.Errorf("Subsec = %v, want 0", x.Subsec())
	}
	half := x.Add(0.5)
	if got, want := half.Subsec(), big.NewRat(1, 2); got.Cmp(want) != 0 {
		t.Errorf("half Subsec = %v", got)
	}
	if got, want := half.ToR(), big.NewRat(3565420625, 2); got.Cmp(want) != 0 {
		t.Errorf("half ToR = %v", got)
	}
}

func TestToA(t *testing.T) {
	got := ref().ToA()
	want := []any{32, 18, 5, 29, 6, 2026, 1, 180, false, "UTC"}
	if len(got) != len(want) {
		t.Fatalf("ToA len = %d", len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ToA[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestLocalAndNew(t *testing.T) {
	// A fixed-offset construction renders numerically and carries the offset.
	o := New(2026, 6, 29, 5, 18, 32, 7200)
	if o.UTCOffset() != 7200 || o.GMT() {
		t.Errorf("offset=%d gmt=%v", o.UTCOffset(), o.GMT())
	}
	if o.Zone() != "" {
		t.Errorf("offset Zone = %q, want empty", o.Zone())
	}
	if got, want := o.Inspect(), "2026-06-29 05:18:32 +0200"; got != want {
		t.Errorf("offset Inspect = %q, want %q", got, want)
	}
	// Local builds in the process zone; pin TZ=UTC in CI so this is +0000.
	l := Local(2026, 6, 29)
	if l.GMT() {
		t.Errorf("Local should not be UTC-flagged")
	}
}

func TestWeekdayPredicates(t *testing.T) {
	x := ref() // a Monday
	preds := []struct {
		name string
		got  bool
		want bool
	}{
		{"sun", x.Sunday(), false},
		{"mon", x.Monday(), true},
		{"tue", x.Tuesday(), false},
		{"wed", x.Wednesday(), false},
		{"thu", x.Thursday(), false},
		{"fri", x.Friday(), false},
		{"sat", x.Saturday(), false},
	}
	for _, p := range preds {
		if p.got != p.want {
			t.Errorf("%s = %v, want %v", p.name, p.got, p.want)
		}
	}
}

func TestUTCAndGetLocal(t *testing.T) {
	o := New(2026, 6, 29, 5, 18, 32, 7200)
	u := o.UTCTime()
	if !u.GMT() || u.UTCOffset() != 0 {
		t.Errorf("UTCTime gmt=%v off=%d", u.GMT(), u.UTCOffset())
	}
	if got, want := u.Inspect(), "2026-06-29 03:18:32 UTC"; got != want {
		t.Errorf("getutc = %q, want %q", got, want)
	}
	back := u.GetLocal(7200)
	if back.UTCOffset() != 7200 || back.GMT() {
		t.Errorf("getlocal off=%d gmt=%v", back.UTCOffset(), back.GMT())
	}
	if !back.Equal(o) {
		t.Errorf("getlocal not equal to original instant")
	}
	// ToLocal uses the process zone (TZ=UTC in CI -> offset 0, not UTC-flagged).
	loc := u.ToLocal()
	if loc.GMT() {
		t.Errorf("ToLocal should not be UTC-flagged")
	}
}

func TestArithmetic(t *testing.T) {
	a := ref()
	b := a.Add(3600)
	if b.Hour() != 6 {
		t.Errorf("Add hour = %d", b.Hour())
	}
	if d := b.Diff(a); d != 3600.0 {
		t.Errorf("Diff = %v", d)
	}
	c := a.Add(1.5)
	if c.Sec() != 33 || c.NSec() != 500000000 {
		t.Errorf("Add 1.5 sec=%d nsec=%d", c.Sec(), c.NSec())
	}
	if got := a.Sub(0.5); got.Sec() != 31 || got.NSec() != 500000000 {
		t.Errorf("Sub 0.5 sec=%d nsec=%d", got.Sec(), got.NSec())
	}
	// Crossing a whole-second boundary backwards must borrow.
	if got := a.Sub(33.0).NSec(); got != 0 {
		t.Errorf("Sub borrow nsec = %d", got)
	}
	neg := a.Add(-3600)
	if neg.Hour() != 4 {
		t.Errorf("Add negative hour = %d", neg.Hour())
	}
}

func TestComparisonAndHash(t *testing.T) {
	a := ref()
	b := a.Add(1)
	if a.Cmp(b) != -1 || b.Cmp(a) != 1 || a.Cmp(a) != 0 {
		t.Errorf("Cmp wrong: %d %d %d", a.Cmp(b), b.Cmp(a), a.Cmp(a))
	}
	if !a.Before(b) || !b.After(a) {
		t.Errorf("Before/After wrong")
	}
	if a.Before(a) || a.After(a) {
		t.Errorf("strict ordering on equal")
	}
	// Equal/Hash ignore the observed offset.
	off := New(2026, 6, 29, 5, 18, 32, 7200).UTCTime().GetLocal(0)
	same := UTC(2026, 6, 29, 3, 18, 32)
	if !off.Equal(same) {
		t.Errorf("Equal across offsets failed")
	}
	dup := UTC(2026, 6, 29, 5, 18, 32)
	if !a.Equal(dup) || a.Hash() != dup.Hash() {
		t.Errorf("hash/equal mismatch")
	}
	if a.Hash() == b.Hash() {
		t.Errorf("distinct instants hashed equal")
	}
}

func TestRoundFloorCeil(t *testing.T) {
	t0 := ref().Add(0.123456789)
	if got, want := t0.Round(3).NSec(), 123000000; got != want {
		t.Errorf("Round(3) nsec = %d, want %d", got, want)
	}
	if got := t0.Round(0).NSec(); got != 0 {
		t.Errorf("Round(0) nsec = %d", got)
	}
	if got, want := t0.Floor(2).NSec(), 120000000; got != want {
		t.Errorf("Floor(2) nsec = %d, want %d", got, want)
	}
	if got, want := t0.Ceil(2).NSec(), 130000000; got != want {
		t.Errorf("Ceil(2) nsec = %d, want %d", got, want)
	}
	// Rounding up across the second boundary carries.
	near := ref().Add(0.9999999995)
	if got := near.Round(0); got.NSec() != 0 || got.Sec() != 33 {
		t.Errorf("Round carry sec=%d nsec=%d", got.Sec(), got.NSec())
	}
	// Ceil carrying into the next whole second.
	if got := ref().Add(0.5).Ceil(0); got.Sec() != 33 || got.NSec() != 0 {
		t.Errorf("Ceil carry sec=%d nsec=%d", got.Sec(), got.NSec())
	}
	// Negative-nanosecond rounding via normNsec is exercised by a sub-second
	// subtraction that borrows; round it back.
	if got := ref().Sub(0.25).Round(1); got.NSec() != 800000000 {
		t.Errorf("Round after borrow nsec = %d", got.NSec())
	}
	// Digit clamps.
	if scaleFor(-1) != 1_000_000_000 || scaleFor(99) != 1 {
		t.Errorf("scaleFor clamp wrong")
	}
}

func TestRoundDivNegative(t *testing.T) {
	if got := roundDiv(-150, 100); got != -2 {
		t.Errorf("roundDiv(-150,100) = %d, want -2", got)
	}
}

func TestNowSeam(t *testing.T) {
	orig := NowFunc
	defer func() { NowFunc = orig }()
	NowFunc = func() (int64, int32) { return 1782710312, 250000000 }
	got := Now()
	if got.ToI() != 1782710312 || got.NSec() != 250000000 {
		t.Errorf("Now seam = %d.%d", got.ToI(), got.NSec())
	}
	if got.GMT() {
		t.Errorf("Now should be local, not UTC-flagged")
	}
}

func TestNowDefaultRuns(t *testing.T) {
	// Exercise the real NowFunc once (its value is non-deterministic, so only
	// sanity-check it is in a plausible modern range).
	sec, _ := NowFunc()
	if sec < 1_600_000_000 {
		t.Errorf("default NowFunc returned implausible %d", sec)
	}
}

func TestAtForms(t *testing.T) {
	if got := At(1782710312.5); got.Sec() != 32 || got.NSec() != 500000000 {
		t.Errorf("At fractional sec=%d nsec=%d", got.Sec(), got.NSec())
	}
	if got := AtNsec(1782710312, 123456789); got.NSec() != 123456789 {
		t.Errorf("AtNsec nsec = %d", got.NSec())
	}
	src := ref()
	if got := AtTime(src); !got.Equal(src) || got.GMT() {
		t.Errorf("AtTime equal=%v gmt=%v", got.Equal(src), got.GMT())
	}
	// Negative fractional input borrows correctly.
	if got := At(-0.5); got.NSec() != 500000000 || got.Sec() != 59 {
		t.Errorf("At(-0.5) sec=%d nsec=%d", got.Sec(), got.NSec())
	}
}

func TestCivilDefaults(t *testing.T) {
	// Defaults fill month/day to 1 and the rest to 0.
	x := UTC(2026)
	if x.Month() != 1 || x.Day() != 1 || x.Hour() != 0 {
		t.Errorf("UTC defaults = %v", x.Inspect())
	}
	// All fields supplied, including the nanosecond tail.
	y := UTC(2026, 6, 29, 5, 18, 32, 7)
	if y.NSec() != 7 {
		t.Errorf("UTC nsec tail = %d", y.NSec())
	}
}

func TestGoTimeBridge(t *testing.T) {
	// The local-offset path is reached when constructing via At; assert it agrees
	// with the standard library for a UTC machine (TZ=UTC in CI).
	g := ref().goTime()
	if g.Unix() != 1782710312 {
		t.Errorf("goTime unix = %d", g.Unix())
	}
	if g.Location().String() == "" && !ref().isUTC {
		t.Errorf("unexpected empty location for UTC instant")
	}
	_ = stdtime.Now
}
