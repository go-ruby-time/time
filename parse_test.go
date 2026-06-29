// Copyright (c) the go-ruby-time/time authors
//
// SPDX-License-Identifier: BSD-3-Clause

package time

import "testing"

func TestStrptimeRoundTrip(t *testing.T) {
	cases := []struct {
		input, layout, wantInspect string
	}{
		{"2026-06-29 05:18:32", "%Y-%m-%d %H:%M:%S", "2026-06-29 05:18:32 +0000"},
		{"29/06/2026", "%d/%m/%Y", "2026-06-29 00:00:00 +0000"},
		{"2026-06-29T05:18:32+0200", "%Y-%m-%dT%H:%M:%S%z", "2026-06-29 05:18:32 +0200"},
		{"2026-06-29T05:18:32Z", "%Y-%m-%dT%H:%M:%S%z", "2026-06-29 05:18:32 UTC"},
		{"1782710312", "%s", "2026-06-29 05:18:32 +0000"},
		{"Jun 29 2026", "%b %d %Y", "2026-06-29 00:00:00 +0000"},
		{"June 29 2026", "%B %d %Y", "2026-06-29 00:00:00 +0000"},
		{"2026-180", "%Y-%j", "2026-06-29 00:00:00 +0000"},
		{"05:18 PM", "%I:%M %p", "1970-01-01 17:18:00 +0000"},
		{"05:18 am", "%I:%M %P", "1970-01-01 05:18:00 +0000"},
		{"Mon 29", "%a %d", "1970-01-29 00:00:00 +0000"},
		{"26-06-29", "%y-%m-%d", "2026-06-29 00:00:00 +0000"},
		{"99-01-01", "%y-%m-%d", "1999-01-01 00:00:00 +0000"},
		{"05:18:32.250", "%H:%M:%S.%L", "1970-01-01 05:18:32.25 +0000"},
		{"05:18:32.123456789", "%H:%M:%S.%N", "1970-01-01 05:18:32.123456789 +0000"},
		{"03", "%k", "1970-01-01 03:00:00 +0000"},
		{"03", "%l", "1970-01-01 03:00:00 +0000"},
		{"05-06:30", "%H%z", "1970-01-01 05:00:00 -0630"},
		{"50%", "%M%%", "1970-01-01 00:50:00 +0000"},
		{"  29", "%e", "1970-01-29 00:00:00 +0000"},
	}
	for _, c := range cases {
		// Layouts without a date part default to the 1970-01-01 epoch day (a
		// deterministic choice; MRI fills in "today" there, which is not).
		got, err := Strptime(c.input, c.layout)
		if err != nil {
			t.Errorf("Strptime(%q,%q) error: %v", c.input, c.layout, err)
			continue
		}
		if got.Inspect() != c.wantInspect {
			t.Errorf("Strptime(%q,%q) = %q, want %q", c.input, c.layout, got.Inspect(), c.wantInspect)
		}
	}
}

func TestStrptimeErrors(t *testing.T) {
	bad := []struct{ input, layout string }{
		{"notayear", "%Y"},
		{"2026-06", "%Y-%m-%dX"}, // literal X missing
		{"x", "%y"},              // non-digit
		{"x", "%m"},              // non-digit month
		{"2026 x", "%Y %d"},      // non-digit day
		{"2026 x", "%Y %H"},      // non-digit hour
		{"2026 x", "%Y %I"},      // non-digit hour12
		{"x", "%M"},              // non-digit min
		{"x", "%S"},              // non-digit sec
		{"x", "%j"},              // non-digit yday
		{"x", "%L"},              // non-digit millis
		{"x", "%N"},              // non-digit nanos
		{"x", "%s"},              // non-digit epoch
		{"XY", "%p"},             // bad meridian
		{"X", "%p"},              // short meridian
		{"Zzz", "%b"},            // bad month name
		{"Zzz", "%a"},            // bad day name
		{"X", "%z"},              // bad offset
		{"+", "%z"},              // truncated offset
		{"X", "%%"},              // literal percent mismatch
		{"X", "%Q"},              // unknown directive
	}
	for _, c := range bad {
		if _, err := Strptime(c.input, c.layout); err == nil {
			t.Errorf("Strptime(%q,%q) expected error", c.input, c.layout)
		}
	}
	// A layout ending in a bare '%' treats it as a literal that must match.
	if _, err := Strptime("%", "%"); err != nil {
		t.Errorf("trailing bare percent literal: %v", err)
	}
}

func TestParse(t *testing.T) {
	cases := []struct{ input, wantInspect string }{
		{"2026-06-29T05:18:32Z", "2026-06-29 05:18:32 UTC"},
		{"2026-06-29T05:18:32.5Z", "2026-06-29 05:18:32.5 UTC"},
		{"2026-06-29 05:18:32 +0530", "2026-06-29 05:18:32 +0530"},
		{"2026-06-29T05:18:32+02:00", "2026-06-29 05:18:32 +0200"},
		{"Mon, 29 Jun 2026 05:18:32 +0000", "2026-06-29 05:18:32 +0000"},
		{"Mon Jun 29 05:18:32 2026", "2026-06-29 05:18:32 +0000"},
		{"  2026-06-29  ", "2026-06-29 00:00:00 +0000"},
	}
	for _, c := range cases {
		got, err := Parse(c.input)
		if err != nil {
			t.Errorf("Parse(%q) error: %v", c.input, err)
			continue
		}
		if got.Inspect() != c.wantInspect {
			t.Errorf("Parse(%q) = %q, want %q", c.input, got.Inspect(), c.wantInspect)
		}
	}
	if _, err := Parse("not a date at all"); err == nil {
		t.Errorf("Parse of garbage should error")
	}
}
