// Copyright (c) the go-ruby-time/time authors
//
// SPDX-License-Identifier: BSD-3-Clause

package time

import (
	"fmt"
	"strconv"
	"strings"
)

// fields accumulates the civil components a strptime scan resolves, defaulting
// to the start of the current epoch day (1970-01-01) for absent parts, as MRI's
// strptime does when a field is omitted.
type fields struct {
	year, month, day     int
	hour, min, sec, nsec int
	yday                 int  // day-of-year via %j, 0 when unused
	pm, havePM           bool // %p marker seen
	hour12               int  // 1..12 from %I, 0 when unused
	offset               int  // seconds east of UTC (%z)
	haveOffset, isUTC    bool // an explicit zone was parsed; Z means UTC
}

// Strptime parses input per the Ruby/C strftime directives in layout, like
// Time.strptime. A mismatch returns an error rather than a Ruby exception (the
// host maps it to ArgumentError).
func Strptime(input, layout string) (*Time, error) {
	f := fields{year: 1970, month: 1, day: 1}
	si := 0 // cursor into input
	for li := 0; li < len(layout); li++ {
		if layout[li] != '%' || li+1 >= len(layout) {
			// Literal: whitespace in the layout matches any run of input space.
			if layout[li] == ' ' {
				for si < len(input) && input[si] == ' ' {
					si++
				}
				continue
			}
			if si >= len(input) || input[si] != layout[li] {
				return nil, fmt.Errorf("time: %q does not match format %q", input, layout)
			}
			si++
			continue
		}
		li++
		if err := f.consume(layout[li], input, &si); err != nil {
			return nil, err
		}
	}
	return f.assemble()
}

// readInt reads up to max digits (at least one) from input at *si.
func readInt(input string, si *int, max int) (int, bool) {
	start := *si
	for *si < len(input) && *si-start < max && input[*si] >= '0' && input[*si] <= '9' {
		*si++
	}
	if *si == start {
		return 0, false
	}
	n, _ := strconv.Atoi(input[start:*si])
	return n, true
}

// skipSpaces advances past any leading spaces (for space-padded fields).
func skipSpaces(input string, si *int) {
	for *si < len(input) && input[*si] == ' ' {
		*si++
	}
}

// consume parses one directive's worth of input, updating the fields, returning
// an error on a mismatch.
func (f *fields) consume(c byte, input string, si *int) error {
	fail := func() error {
		return fmt.Errorf("time: cannot parse %%%c at %q", c, input[min(*si, len(input)):])
	}
	switch c {
	case 'Y':
		n, ok := readInt(input, si, 4)
		if !ok {
			return fail()
		}
		f.year = n
	case 'y':
		n, ok := readInt(input, si, 2)
		if !ok {
			return fail()
		}
		if n < 69 {
			f.year = 2000 + n
		} else {
			f.year = 1900 + n
		}
	case 'm':
		n, ok := readInt(input, si, 2)
		if !ok {
			return fail()
		}
		f.month = n
	case 'd', 'e':
		skipSpaces(input, si)
		n, ok := readInt(input, si, 2)
		if !ok {
			return fail()
		}
		f.day = n
	case 'H', 'k':
		skipSpaces(input, si)
		n, ok := readInt(input, si, 2)
		if !ok {
			return fail()
		}
		f.hour = n
	case 'I', 'l':
		skipSpaces(input, si)
		n, ok := readInt(input, si, 2)
		if !ok {
			return fail()
		}
		f.hour12 = n
	case 'M':
		n, ok := readInt(input, si, 2)
		if !ok {
			return fail()
		}
		f.min = n
	case 'S':
		n, ok := readInt(input, si, 2)
		if !ok {
			return fail()
		}
		f.sec = n
	case 'j':
		n, ok := readInt(input, si, 3)
		if !ok {
			return fail()
		}
		f.yday = n
	case 'L':
		n, ok := readInt(input, si, 3)
		if !ok {
			return fail()
		}
		f.nsec = n * 1_000_000
	case 'N':
		start := *si
		n, ok := readInt(input, si, 9)
		if !ok {
			return fail()
		}
		digits := *si - start
		for digits < 9 {
			n *= 10
			digits++
		}
		f.nsec = n
	case 's':
		n, ok := readInt(input, si, 19)
		if !ok {
			return fail()
		}
		return f.fromEpoch(int64(n))
	case 'p', 'P':
		if *si+2 > len(input) {
			return fail()
		}
		switch strings.ToUpper(input[*si : *si+2]) {
		case "AM":
			f.havePM, f.pm = true, false
		case "PM":
			f.havePM, f.pm = true, true
		default:
			return fail()
		}
		*si += 2
	case 'b', 'h', 'B':
		if !f.parseMonthName(input, si) {
			return fail()
		}
	case 'a', 'A':
		if !skipDayName(input, si) {
			return fail()
		}
	case 'z':
		if !f.parseOffset(input, si) {
			return fail()
		}
	case '%':
		if *si >= len(input) || input[*si] != '%' {
			return fail()
		}
		*si++
	default:
		return fail()
	}
	return nil
}

// fromEpoch resolves a whole %s timestamp into civil fields at UTC.
func (f *fields) fromEpoch(sec int64) error {
	t := AtNsec(sec, 0).UTCTime()
	f.year, f.month, f.day = t.Year(), t.Month(), t.Day()
	f.hour, f.min, f.sec = t.Hour(), t.Min(), t.Sec()
	f.haveOffset, f.isUTC, f.offset = true, false, 0
	return nil
}

// parseMonthName matches a 3+ letter month name (case-insensitive), setting month.
func (f *fields) parseMonthName(input string, si *int) bool {
	for i, full := range monthNames {
		for _, name := range []string{full, full[:3]} {
			if len(input)-*si >= len(name) && strings.EqualFold(input[*si:*si+len(name)], name) {
				f.month = i + 1
				*si += len(name)
				return true
			}
		}
	}
	return false
}

// skipDayName consumes a weekday name (%a/%A); the value is unused by assembly.
func skipDayName(input string, si *int) bool {
	for _, full := range dayNames {
		for _, name := range []string{full, full[:3]} {
			if len(input)-*si >= len(name) && strings.EqualFold(input[*si:*si+len(name)], name) {
				*si += len(name)
				return true
			}
		}
	}
	return false
}

// parseOffset matches Z, ±HH, ±HHMM or ±HH:MM and records the zone.
func (f *fields) parseOffset(input string, si *int) bool {
	if *si < len(input) && (input[*si] == 'Z' || input[*si] == 'z') {
		f.haveOffset, f.isUTC, f.offset = true, true, 0
		*si++
		return true
	}
	if *si >= len(input) || (input[*si] != '+' && input[*si] != '-') {
		return false
	}
	sign := 1
	if input[*si] == '-' {
		sign = -1
	}
	*si++
	h, ok := readInt(input, si, 2)
	if !ok {
		return false
	}
	if *si < len(input) && input[*si] == ':' {
		*si++
	}
	m, _ := readInt(input, si, 2)
	f.haveOffset, f.isUTC = true, false
	f.offset = sign * (h*3600 + m*60)
	return true
}

// assemble materialises the accumulated civil fields into a Time, resolving the
// 12-hour clock and an absent zone (which defaults to the local zone, as MRI does).
func (f *fields) assemble() (*Time, error) {
	hour := f.hour
	if f.hour12 != 0 {
		hour = f.hour12 % 12
		if f.havePM && f.pm {
			hour += 12
		}
	}
	if f.yday != 0 {
		// Resolve %j: day-of-year within f.year.
		t := UTC(f.year, 1, 1).Add(float64((f.yday - 1) * 86400))
		f.month, f.day = t.Month(), t.Day()
	}
	if f.haveOffset {
		return fromCivil(f.year, f.month, f.day, hour, f.min, f.sec, f.nsec, f.offset, f.isUTC), nil
	}
	// No zone given: observe in the local zone, like MRI.
	loc := Local(f.year, f.month, f.day, hour, f.min, f.sec, f.nsec)
	return loc, nil
}

// Parse parses a date/time string heuristically, like Time.parse from the `time`
// stdlib: it tries the common ISO-8601, RFC-2822 and space-separated layouts. An
// unparseable string returns an error.
func Parse(input string) (*Time, error) {
	s := strings.TrimSpace(input)
	layouts := []string{
		"%Y-%m-%dT%H:%M:%S.%N%z",
		"%Y-%m-%dT%H:%M:%S%z",
		"%Y-%m-%dT%H:%M:%S",
		"%Y-%m-%d %H:%M:%S.%N %z",
		"%Y-%m-%d %H:%M:%S %z",
		"%Y-%m-%d %H:%M:%S",
		"%a, %d %b %Y %H:%M:%S %z",
		"%a %b %e %H:%M:%S %Y",
		"%Y-%m-%d",
	}
	for _, l := range layouts {
		if t, err := Strptime(s, l); err == nil {
			return t, nil
		}
	}
	return nil, fmt.Errorf("time: cannot parse %q", input)
}
