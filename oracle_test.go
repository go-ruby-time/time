// Copyright (c) the go-ruby-time/time authors
//
// SPDX-License-Identifier: BSD-3-Clause

package time

import (
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

// rubyBin locates a usable `ruby` once and skips the differential oracle when it
// is absent (the Windows lane and the qemu cross-arch lanes have no target ruby),
// so the deterministic suite alone drives the 100% gate there. The oracle is also
// gated on MRI >= 4.0 so older interpreters with divergent Time#inspect formatting
// do not produce spurious mismatches.
func rubyBin(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("ruby")
	if err != nil {
		t.Skip("ruby not on PATH; skipping MRI oracle")
	}
	ver := strings.TrimSpace(runRuby(t, path, `print RUBY_VERSION`))
	if ver < "4.0" {
		t.Skipf("ruby %s < 4.0; skipping MRI oracle", ver)
	}
	return path
}

// runRuby executes a Ruby script under `ruby -rtime` and returns its stdout. The
// script binmodes stdin/stdout so Windows text-mode never rewrites the bytes (the
// go-ruby-erb lesson); the oracle always runs in the UTC zone for determinism.
func runRuby(t *testing.T, bin, script string) string {
	t.Helper()
	cmd := exec.Command(bin, "-rtime", "-e",
		"$stdout.binmode\n$stdin.binmode\n"+script)
	cmd.Env = append(cmd.Environ(), "TZ=UTC")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("ruby error: %v\nscript:\n%s\noutput:\n%s", err, script, out)
	}
	return string(out)
}

// TestOracleFormatting checks every formatter against MRI for the reference
// instant in UTC and at a fixed offset.
func TestOracleFormatting(t *testing.T) {
	bin := rubyBin(t)
	cases := []struct {
		name   string
		got    string
		script string
	}{
		{"utc_inspect", ref().Inspect(), `print Time.utc(2026,6,29,5,18,32).inspect`},
		{"utc_to_s", ref().ToS(), `print Time.utc(2026,6,29,5,18,32).to_s`},
		{"utc_ctime", ref().CTime(), `print Time.utc(2026,6,29,5,18,32).ctime`},
		{"utc_iso", ref().ISO8601(), `print Time.utc(2026,6,29,5,18,32).iso8601`},
		{"utc_rfc", ref().RFC2822(), `print Time.utc(2026,6,29,5,18,32).rfc2822`},
		{"utc_http", ref().HTTPDate(), `print Time.utc(2026,6,29,5,18,32).httpdate`},
		{"off_inspect", New(2026, 6, 29, 5, 18, 32, 7200).Inspect(),
			`print Time.new(2026,6,29,5,18,32,"+02:00").inspect`},
		{"off_iso", New(2026, 6, 29, 5, 18, 32, 7200).ISO8601(),
			`print Time.new(2026,6,29,5,18,32,"+02:00").iso8601`},
		{"off_http", New(2026, 6, 29, 5, 18, 32, 7200).HTTPDate(),
			`print Time.new(2026,6,29,5,18,32,"+02:00").httpdate`},
		{"frac_inspect", ref().Add(0.5).Inspect(),
			`print (Time.utc(2026,6,29,5,18,32)+0.5).inspect`},
		{"frac_iso3", ref().Add(0.5).ISO8601(3),
			`print (Time.utc(2026,6,29,5,18,32)+0.5).iso8601(3)`},
	}
	for _, c := range cases {
		want := runRuby(t, bin, c.script)
		if c.got != want {
			t.Errorf("%s: Go = %q, MRI = %q", c.name, c.got, want)
		}
	}
}

// TestOracleStrftime checks a wide directive corpus against MRI.
func TestOracleStrftime(t *testing.T) {
	bin := rubyBin(t)
	directives := []string{
		"%Y-%m-%d", "%H:%M:%S", "%A %B %d", "%a %b %e", "%j", "%p %I:%M%P",
		"%Z", "%y", "%e", "%c", "%x", "%X", "%F", "%T", "%R", "%s", "%L",
		"%N", "%3N", "%6N", "%u", "%w", "%G", "%V", "%U", "%W", "%C", "%D",
		"%-d", "%-m", "%h", "%k", "%l", "%:z", "%::z", "%z", "%r",
	}
	for _, d := range directives {
		got := ref().Strftime(d)
		want := runRuby(t, bin,
			`print Time.utc(2026,6,29,5,18,32).strftime(`+rubyStr(d)+`)`)
		if got != want {
			t.Errorf("strftime(%q): Go = %q, MRI = %q", d, got, want)
		}
	}
}

// TestOracleComponentsAndArith cross-checks numeric accessors and arithmetic.
func TestOracleComponentsAndArith(t *testing.T) {
	bin := rubyBin(t)
	x := ref()
	pairs := []struct {
		got    string
		script string
	}{
		{itoa(x.ToI()), `print Time.utc(2026,6,29,5,18,32).to_i`},
		{itoa(int64(x.YDay())), `print Time.utc(2026,6,29,5,18,32).yday`},
		{itoa(int64(x.WDay())), `print Time.utc(2026,6,29,5,18,32).wday`},
		{itoa(int64(x.UTCOffset())), `print Time.utc(2026,6,29,5,18,32).utc_offset`},
		{itoa(x.Add(3600).ToI()), `print (Time.utc(2026,6,29,5,18,32)+3600).to_i`},
		{ftoa(x.Add(1).Diff(x)), `print((Time.utc(2026,6,29,5,18,33))-(Time.utc(2026,6,29,5,18,32)))`},
	}
	for _, p := range pairs {
		want := strings.TrimSpace(runRuby(t, bin, p.script))
		if p.got != want {
			t.Errorf("Go = %q, MRI = %q (script %q)", p.got, want, p.script)
		}
	}
}

// TestOracleParse checks Parse / Strptime agree with MRI's Time.parse / strptime
// for zone-bearing inputs (so the result is independent of the machine zone).
func TestOracleParse(t *testing.T) {
	bin := rubyBin(t)
	cases := []struct {
		got    string
		script string
	}{
		{mustParse(t, "2026-06-29T05:18:32Z").Inspect(),
			`print Time.parse("2026-06-29T05:18:32Z").inspect`},
		{mustParse(t, "2026-06-29 05:18:32 +0530").Inspect(),
			`print Time.parse("2026-06-29 05:18:32 +0530").inspect`},
		{mustStrptime(t, "2026-06-29T05:18:32+0200", "%Y-%m-%dT%H:%M:%S%z").Inspect(),
			`print Time.strptime("2026-06-29T05:18:32+0200","%Y-%m-%dT%H:%M:%S%z").inspect`},
		{mustStrptime(t, "1782710312", "%s").Inspect(),
			`print Time.strptime("1782710312","%s").inspect`},
	}
	for _, c := range cases {
		want := runRuby(t, bin, c.script)
		if c.got != want {
			t.Errorf("Go = %q, MRI = %q (script %q)", c.got, want, c.script)
		}
	}
}

// --- oracle test helpers ----------------------------------------------------

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
func ftoa(f float64) string {
	s := strconv.FormatFloat(f, 'f', -1, 64)
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return s
}

func mustParse(t *testing.T, s string) *Time {
	t.Helper()
	v, err := Parse(s)
	if err != nil {
		t.Fatalf("Parse(%q): %v", s, err)
	}
	return v
}

func mustStrptime(t *testing.T, s, layout string) *Time {
	t.Helper()
	v, err := Strptime(s, layout)
	if err != nil {
		t.Fatalf("Strptime(%q,%q): %v", s, layout, err)
	}
	return v
}

// rubyStr renders a Go string as a Ruby double-quoted literal for the scripts.
func rubyStr(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return `"` + r.Replace(s) + `"`
}
