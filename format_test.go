// Copyright (c) the go-ruby-time/time authors
//
// SPDX-License-Identifier: BSD-3-Clause

package time

import "testing"

// These golden expectations are the verbatim output of MRI 4.0.5 for the fixed
// reference instant (see time_test.go), captured with `ruby -rtime -e`.
func TestFormattersGolden(t *testing.T) {
	x := ref()
	cases := []struct{ name, got, want string }{
		{"to_s", x.ToS(), "2026-06-29 05:18:32 UTC"},
		{"inspect", x.Inspect(), "2026-06-29 05:18:32 UTC"},
		{"ctime", x.CTime(), "Mon Jun 29 05:18:32 2026"},
		{"iso8601", x.ISO8601(), "2026-06-29T05:18:32Z"},
		{"rfc2822", x.RFC2822(), "Mon, 29 Jun 2026 05:18:32 -0000"},
		{"httpdate", x.HTTPDate(), "Mon, 29 Jun 2026 05:18:32 GMT"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.name, c.got, c.want)
		}
	}
}

func TestFormattersWithOffset(t *testing.T) {
	o := New(2026, 6, 29, 5, 18, 32, 7200)
	cases := []struct{ name, got, want string }{
		{"to_s", o.ToS(), "2026-06-29 05:18:32 +0200"},
		{"inspect", o.Inspect(), "2026-06-29 05:18:32 +0200"},
		{"ctime", o.CTime(), "Mon Jun 29 05:18:32 2026"},
		{"iso8601", o.ISO8601(), "2026-06-29T05:18:32+02:00"},
		{"rfc2822", o.RFC2822(), "Mon, 29 Jun 2026 05:18:32 +0200"},
		{"httpdate", o.HTTPDate(), "Mon, 29 Jun 2026 03:18:32 GMT"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.name, c.got, c.want)
		}
	}
	// A negative offset exercises the sign branch of offsetParts.
	w := New(2026, 6, 29, 5, 18, 32, -18000)
	if got, want := w.Inspect(), "2026-06-29 05:18:32 -0500"; got != want {
		t.Errorf("negative offset inspect = %q, want %q", got, want)
	}
}

func TestSubsecondFormatting(t *testing.T) {
	half := ref().Add(0.5)
	if got, want := half.Inspect(), "2026-06-29 05:18:32.5 UTC"; got != want {
		t.Errorf("subsec inspect = %q, want %q", got, want)
	}
	if got, want := half.ToS(), "2026-06-29 05:18:32 UTC"; got != want {
		t.Errorf("subsec to_s = %q, want %q", got, want)
	}
	if got, want := half.ISO8601(3), "2026-06-29T05:18:32.500Z"; got != want {
		t.Errorf("iso8601(3) = %q, want %q", got, want)
	}
	nano := UTC(2026, 1, 1).Add(0.123456789)
	if got, want := nano.Inspect(), "2026-01-01 00:00:00.123456789 UTC"; got != want {
		t.Errorf("nano inspect = %q, want %q", got, want)
	}
	// iso8601 with more than 9 requested digits pads with trailing zeros.
	if got, want := nano.ISO8601(12), "2026-01-01T00:00:00.123456789000Z"; got != want {
		t.Errorf("iso8601(12) = %q, want %q", got, want)
	}
	// iso8601 with no fractional argument drops the sub-second part.
	if got, want := nano.ISO8601(), "2026-01-01T00:00:00Z"; got != want {
		t.Errorf("iso8601() = %q, want %q", got, want)
	}
}

func TestStrftimeDirectives(t *testing.T) {
	x := ref()
	cases := []struct{ format, want string }{
		{"%Y-%m-%d", "2026-06-29"},
		{"%H:%M:%S", "05:18:32"},
		{"%Y-%m-%dT%H:%M:%S%z", "2026-06-29T05:18:32+0000"},
		{"%A %B %d", "Monday June 29"},
		{"%a %b %e", "Mon Jun 29"},
		{"%j", "180"},
		{"%p %I:%M%P", "AM 05:18am"},
		{"%%", "%"},
		{"%Z", "UTC"},
		{"%y", "26"},
		{"%e", "29"},
		{"%c", "Mon Jun 29 05:18:32 2026"},
		{"%x", "06/29/26"},
		{"%X", "05:18:32"},
		{"%F", "2026-06-29"},
		{"%T", "05:18:32"},
		{"%R", "05:18"},
		{"%s", "1782710312"},
		{"%L", "000"},
		{"%N", "000000000"},
		{"%3N", "000"},
		{"%6N", "000000"},
		{"%9N", "000000000"},
		{"%u", "1"},
		{"%w", "1"},
		{"%G", "2026"},
		{"%V", "27"},
		{"%U", "26"},
		{"%W", "26"},
		{"%C", "20"},
		{"%D", "06/29/26"},
		{"%-d", "29"},
		{"%-m", "6"},
		{"%h", "Jun"},
		{"%k", " 5"},
		{"%l", " 5"},
		{"%n", "\n"},
		{"%t", "\t"},
		{"%:z", "+00:00"},
		{"%::z", "+00:00:00"},
		{"%r", "05:18:32 AM"},
		{"literal text", "literal text"},
		{"%Q", "%Q"},     // unknown directive echoes verbatim
		{"100%", "100%"}, // trailing bare percent
	}
	for _, c := range cases {
		if got := x.Strftime(c.format); got != c.want {
			t.Errorf("Strftime(%q) = %q, want %q", c.format, got, c.want)
		}
	}
}

func TestStrftimePaddingAndPM(t *testing.T) {
	y := UTC(2026, 1, 5, 3, 4, 9)
	if got, want := y.Strftime("%e|%k|%l|%-d|%_d|%3N"), " 5| 3| 3|5| 5|000"; got != want {
		t.Errorf("padding = %q, want %q", got, want)
	}
	// PM marker and 12-hour wrap at noon/midnight.
	noon := UTC(2026, 1, 1, 12, 0, 0)
	if got, want := noon.Strftime("%I %p %P %l"), "12 PM pm 12"; got != want {
		t.Errorf("noon = %q, want %q", got, want)
	}
	mid := UTC(2026, 1, 1, 0, 0, 0)
	if got, want := mid.Strftime("%I %p"), "12 AM"; got != want {
		t.Errorf("midnight = %q, want %q", got, want)
	}
	// A non-UTC instant's %Z is empty (MRI's nil zone) and %z is numeric.
	o := New(2026, 6, 29, 5, 18, 32, -18000)
	if got, want := o.Strftime("[%Z]%z%:z%::z"), "[]-0500-05:00-05:00:00"; got != want {
		t.Errorf("offset zone directives = %q, want %q", got, want)
	}
}

func TestStrftimeSundayISOWeekday(t *testing.T) {
	// 2026-06-28 is a Sunday: %u must map it to 7 (not 0), %w stays 0.
	sun := UTC(2026, 6, 28)
	if got, want := sun.Strftime("%u %w"), "7 0"; got != want {
		t.Errorf("Sunday %%u/%%w = %q, want %q", got, want)
	}
}

func TestStrftimeTrailingFlag(t *testing.T) {
	// A format ending in a flag with no conversion letter echoes the tail.
	if got := ref().Strftime("%-"); got != "%-" {
		t.Errorf("trailing flag = %q, want %q", got, "%-")
	}
	if got := ref().Strftime("%0"); got != "%0" {
		t.Errorf("trailing width = %q, want %q", got, "%0")
	}
}
