// Copyright (c) the go-ruby-time/time authors
//
// SPDX-License-Identifier: BSD-3-Clause

package time

import (
	"fmt"
	"strconv"
	"strings"
	stdtime "time"
)

var monthNames = []string{
	"January", "February", "March", "April", "May", "June",
	"July", "August", "September", "October", "November", "December",
}

var dayNames = []string{
	"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday",
}

// pad2 zero-pads n to two digits.
func pad2(n int) string { return fmt.Sprintf("%02d", n) }

// offsetParts decomposes the receiver's UTC offset into sign and h/m/s magnitudes.
func (t *Time) offsetParts() (sign string, h, m, s int) {
	off := t.offset
	sign = "+"
	if off < 0 {
		sign = "-"
		off = -off
	}
	return sign, off / 3600, (off % 3600) / 60, off % 60
}

// offsetColon renders the offset as ±HH:MM (the iso8601 / "%:z" form).
func (t *Time) offsetColon() string {
	sign, h, m, _ := t.offsetParts()
	return fmt.Sprintf("%s%02d:%02d", sign, h, m)
}

// offsetNoColon renders the offset as ±HHMM (the "%z" / inspect / to_s form).
func (t *Time) offsetNoColon() string {
	sign, h, m, _ := t.offsetParts()
	return fmt.Sprintf("%s%02d%02d", sign, h, m)
}

// fracString renders the sub-second nanoseconds as a "." + trimmed-fraction, or
// "" when whole-second — the form Time#inspect appends.
func (t *Time) fracString() string {
	if t.nsec == 0 {
		return ""
	}
	s := fmt.Sprintf("%09d", t.nsec)
	s = strings.TrimRight(s, "0")
	return "." + s
}

// ToS renders MRI's Time#to_s ("2026-06-29 05:18:32 UTC" / "... +0200"). It
// never shows sub-seconds, matching MRI.
func (t *Time) ToS() string {
	g := t.goTime()
	zone := t.offsetNoColon()
	if t.isUTC {
		zone = "UTC"
	}
	return fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d %s",
		g.Year(), int(g.Month()), g.Day(), g.Hour(), g.Minute(), g.Second(), zone)
}

// Inspect renders MRI 4.0's Time#inspect, which is to_s plus a trimmed
// sub-second fraction when nsec is non-zero.
func (t *Time) Inspect() string {
	g := t.goTime()
	zone := t.offsetNoColon()
	if t.isUTC {
		zone = "UTC"
	}
	return fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d%s %s",
		g.Year(), int(g.Month()), g.Day(), g.Hour(), g.Minute(), g.Second(),
		t.fracString(), zone)
}

// CTime renders Time#ctime / #asctime ("Mon Jun 29 05:18:32 2026").
func (t *Time) CTime() string {
	g := t.goTime()
	return fmt.Sprintf("%s %s %2d %02d:%02d:%02d %04d",
		dayNames[g.Weekday()][:3], monthNames[g.Month()-1][:3], g.Day(),
		g.Hour(), g.Minute(), g.Second(), g.Year())
}

// ISO8601 renders Time#iso8601 / #xmlschema, optionally with `fraction` decimal
// places of sub-second precision, and the zone as Z (UTC) or ±HH:MM.
func (t *Time) ISO8601(fraction ...int) string {
	g := t.goTime()
	base := fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02d",
		g.Year(), int(g.Month()), g.Day(), g.Hour(), g.Minute(), g.Second())
	if len(fraction) > 0 && fraction[0] > 0 {
		base += "." + t.fracDigits(fraction[0])
	}
	if t.isUTC {
		return base + "Z"
	}
	return base + t.offsetColon()
}

// fracDigits renders the sub-second part to exactly n decimal places.
func (t *Time) fracDigits(n int) string {
	s := fmt.Sprintf("%09d", t.nsec)
	if n <= 9 {
		return s[:n]
	}
	return s + strings.Repeat("0", n-9)
}

// RFC2822 renders Time#rfc2822 / #rfc822 ("Mon, 29 Jun 2026 05:18:32 +0000"). A
// UTC instant uses the conventional "-0000" zone MRI emits.
func (t *Time) RFC2822() string {
	g := t.goTime()
	zone := t.offsetNoColon()
	if t.isUTC {
		zone = "-0000"
	}
	return fmt.Sprintf("%s, %02d %s %04d %02d:%02d:%02d %s",
		dayNames[g.Weekday()][:3], g.Day(), monthNames[g.Month()-1][:3], g.Year(),
		g.Hour(), g.Minute(), g.Second(), zone)
}

// HTTPDate renders Time#httpdate ("Mon, 29 Jun 2026 05:18:32 GMT"), always in GMT.
func (t *Time) HTTPDate() string {
	g := t.UTCTime().goTime()
	return fmt.Sprintf("%s, %02d %s %04d %02d:%02d:%02d GMT",
		dayNames[g.Weekday()][:3], g.Day(), monthNames[g.Month()-1][:3], g.Year(),
		g.Hour(), g.Minute(), g.Second())
}

// Strftime formats the instant per the Ruby/C strftime directives, like
// Time#strftime. Unknown directives pass through verbatim with their leading '%'.
func (t *Time) Strftime(format string) string {
	var b strings.Builder
	for i := 0; i < len(format); i++ {
		if format[i] != '%' || i+1 >= len(format) {
			b.WriteByte(format[i])
			continue
		}
		// "%:z" and "%::z" are colon-offset variants handled up front.
		if rest := format[i+1:]; strings.HasPrefix(rest, ":z") {
			b.WriteString(t.offsetColon())
			i += 2
			continue
		} else if strings.HasPrefix(rest, "::z") {
			sign, h, m, s := t.offsetParts()
			b.WriteString(fmt.Sprintf("%s%02d:%02d:%02d", sign, h, m, s))
			i += 3
			continue
		}
		// Collect an optional flag and width before the conversion letter.
		j := i + 1
		flag := byte(0)
		for j < len(format) && (format[j] == '-' || format[j] == '_' || format[j] == '0') {
			flag = format[j]
			j++
		}
		width := 0
		for j < len(format) && format[j] >= '0' && format[j] <= '9' {
			width = width*10 + int(format[j]-'0')
			j++
		}
		if j >= len(format) {
			b.WriteString(format[i:])
			break
		}
		out, ok := t.directive(format[j], flag, width)
		if !ok {
			b.WriteString(format[i : j+1])
		} else {
			b.WriteString(out)
		}
		i = j
	}
	return b.String()
}

// applyFlag re-pads a fixed-width numeric field per the strftime flag: '-' drops
// padding, '_' pads with spaces, '0' (and the default) pads with zeros.
func applyFlag(numeric int, flag byte, defWidth int) string {
	s := strconv.Itoa(numeric)
	switch flag {
	case '-':
		return s
	case '_':
		for len(s) < defWidth {
			s = " " + s
		}
		return s
	default:
		for len(s) < defWidth {
			s = "0" + s
		}
		return s
	}
}

// directive renders a single strftime conversion. ok is false for an unknown
// letter so the caller can echo it literally.
func (t *Time) directive(c, flag byte, width int) (string, bool) {
	g := t.goTime()
	switch c {
	case 'Y':
		return fmt.Sprintf("%04d", g.Year()), true
	case 'C':
		return pad2(g.Year() / 100), true
	case 'y':
		return pad2(g.Year() % 100), true
	case 'm':
		return applyFlag(int(g.Month()), flag, 2), true
	case 'd':
		return applyFlag(g.Day(), flag, 2), true
	case 'e':
		return fmt.Sprintf("%2d", g.Day()), true
	case 'H':
		return applyFlag(g.Hour(), flag, 2), true
	case 'k':
		return fmt.Sprintf("%2d", g.Hour()), true
	case 'I':
		return pad2(hour12(g.Hour())), true
	case 'l':
		return fmt.Sprintf("%2d", hour12(g.Hour())), true
	case 'M':
		return pad2(g.Minute()), true
	case 'S':
		return pad2(g.Second()), true
	case 'p':
		return ampm(g.Hour(), true), true
	case 'P':
		return ampm(g.Hour(), false), true
	case 'A':
		return dayNames[g.Weekday()], true
	case 'a':
		return dayNames[g.Weekday()][:3], true
	case 'B':
		return monthNames[g.Month()-1], true
	case 'b', 'h':
		return monthNames[g.Month()-1][:3], true
	case 'j':
		return fmt.Sprintf("%03d", g.YearDay()), true
	case 'w':
		return strconv.Itoa(int(g.Weekday())), true
	case 'u':
		wd := int(g.Weekday())
		if wd == 0 {
			wd = 7
		}
		return strconv.Itoa(wd), true
	case 'U':
		return pad2(weekOfYear(g, false)), true
	case 'W':
		return pad2(weekOfYear(g, true)), true
	case 'V':
		_, wk := g.ISOWeek()
		return pad2(wk), true
	case 'G':
		yr, _ := g.ISOWeek()
		return fmt.Sprintf("%04d", yr), true
	case 'Z':
		if t.isUTC {
			return "UTC", true
		}
		return "", true
	case 'z':
		return t.offsetNoColon(), true
	case 's':
		return strconv.FormatInt(t.sec, 10), true
	case 'L':
		return t.fracDigits(3), true
	case 'N':
		if width == 0 {
			width = 9
		}
		return t.fracDigits(width), true
	case 'c':
		return t.CTime(), true
	case 'x', 'D':
		return fmt.Sprintf("%02d/%02d/%02d", int(g.Month()), g.Day(), g.Year()%100), true
	case 'X', 'T':
		return fmt.Sprintf("%02d:%02d:%02d", g.Hour(), g.Minute(), g.Second()), true
	case 'F':
		return fmt.Sprintf("%04d-%02d-%02d", g.Year(), int(g.Month()), g.Day()), true
	case 'R':
		return fmt.Sprintf("%02d:%02d", g.Hour(), g.Minute()), true
	case 'r':
		return fmt.Sprintf("%02d:%02d:%02d %s", hour12(g.Hour()), g.Minute(), g.Second(), ampm(g.Hour(), true)), true
	case 'n':
		return "\n", true
	case 't':
		return "\t", true
	case '%':
		return "%", true
	}
	return "", false
}

// hour12 maps a 0..23 hour to the 1..12 clock used by %I / %l.
func hour12(h int) int {
	h %= 12
	if h == 0 {
		return 12
	}
	return h
}

// ampm returns the AM/PM marker, upper-cased for %p and lower for %P.
func ampm(hour int, upper bool) string {
	s := "am"
	if hour >= 12 {
		s = "pm"
	}
	if upper {
		return strings.ToUpper(s)
	}
	return s
}

// weekOfYear computes the %U (Sunday-start) or %W (Monday-start) week number:
// the week containing the first such start-day is week 1, days before it week 0.
func weekOfYear(g stdtime.Time, mondayStart bool) int {
	yday := g.YearDay() - 1 // 0-based day index in the year
	wday := int(g.Weekday())
	if mondayStart {
		// Shift so Monday=0 … Sunday=6.
		wday = (wday + 6) % 7
	}
	return (yday + 7 - wday) / 7
}
