<p align="center"><img src="https://raw.githubusercontent.com/go-ruby-time/brand/main/social/go-ruby-time-time.png" alt="go-ruby-time/time" width="720"></p>

# time — go-ruby-time

[![Docs](https://img.shields.io/badge/docs-mkdocs--material-DC2626)](https://go-ruby-time.github.io/docs/)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A pure-Go (no cgo) reimplementation of Ruby's [`Time`](https://docs.ruby-lang.org/en/master/Time.html)
class and the [`time`](https://docs.ruby-lang.org/en/master/Time.html) standard
library's parsing extensions** — the epoch-based instant distinct from `Date`,
matching MRI 4.0.5's API surface and formatting byte-for-byte. It builds, formats,
parses, and does arithmetic over instants **without any Ruby runtime**, using Go's
`time` package only for the underlying civil/zone math.

It is the `Time` backend for
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby), but is a
**standalone, reusable** module with no dependency on the Ruby runtime — a sibling
of [go-ruby-date](https://github.com/go-ruby-date/date) and
[go-ruby-yaml](https://github.com/go-ruby-yaml/yaml).

> **Deterministic by construction.** The only non-deterministic input — the wall
> clock behind `Time.now` — is isolated behind a single package seam (`NowFunc`)
> a test can pin. Every other operation is a pure function of its inputs, so the
> whole suite (which pins its own process zone to UTC) is reproducible on any
> machine.

## Features

A faithful port of `Time`, validated against the `ruby` binary on every platform:

- **Constructors** — `Now`, `At` / `AtNsec` / `AtTime` (`Time.at`), `UTC`
  (`Time.utc` / `gm`), `Local` (`Time.local` / `mktime`), `New` (`Time.new` with a
  fixed UTC offset), plus `Parse` (`Time.parse`) and `Strptime` (`Time.strptime`).
- **Components** — `Year` / `Month` / `Day` / `Hour` / `Min` / `Sec` / `USec` /
  `NSec` / `Subsec` / `WDay` / `YDay` / `UTCOffset` / `Zone` / `GMT` / `DST`, the
  weekday predicates (`Monday` … `Sunday`), and the `to_i` / `to_f` / `to_r` /
  `to_a` conversions.
- **Zone shifts** — `UTCTime` (`getutc` / `utc` / `gmtime`), `GetLocal`
  (`getlocal(offset)`), and `ToLocal` (`localtime`), preserving the instant while
  changing the observed offset.
- **Arithmetic** — `Add` / `Sub` shift by (fractional) seconds; `Diff` is
  `Time - Time` → a float; `Round` / `Floor` / `Ceil` to any sub-second precision.
- **Ordering** — `Cmp` (`<=>`), `Before` / `After`, `Equal` (`==` / `eql?`), and a
  `Hash` consistent with `Equal` — all offset-independent (only the instant matters).
- **Formatting** — `Strftime` (the full C/Ruby directive set, including `%-`/`%_`
  flags, `%N`/`%L` sub-second, ISO week `%V`/`%G`, and the `%z`/`%:z`/`%::z` offset
  forms), plus `Inspect`, `ToS`, `CTime`, `ISO8601` / xmlschema, `RFC2822`, and
  `HTTPDate` — each matching MRI's exact bytes (e.g. `inspect` shows a trimmed
  sub-second fraction, `to_s` never does; a UTC `rfc2822` zone is `-0000`).

CGO-free, dependency-free, **100% test coverage**, `gofmt` + `go vet` clean, and
green across the six 64-bit Go targets (amd64, arm64, riscv64, loong64, ppc64le,
s390x).

## Install

```sh
go get github.com/go-ruby-time/time
```

## Usage

```go
package main

import (
	"fmt"

	rtime "github.com/go-ruby-time/time"
)

func main() {
	t := rtime.UTC(2026, 6, 29, 5, 18, 32) // Time.utc(2026,6,29,5,18,32)

	fmt.Println(t.Inspect())          // 2026-06-29 05:18:32 UTC
	fmt.Println(t.Strftime("%A, %d %B %Y")) // Monday, 29 June 2026
	fmt.Println(t.ISO8601())          // 2026-06-29T05:18:32Z
	fmt.Println(t.RFC2822())          // Mon, 29 Jun 2026 05:18:32 -0000
	fmt.Println(t.Add(90).Inspect())  // 2026-06-29 05:20:02 UTC

	off := rtime.New(2026, 6, 29, 5, 18, 32, 2*3600) // Time.new(..., "+02:00")
	fmt.Println(off.Inspect())        // 2026-06-29 05:18:32 +0200
	fmt.Println(off.UTCTime().Inspect()) // 2026-06-29 03:18:32 UTC

	p, _ := rtime.Parse("2026-06-29T05:18:32+05:30")
	fmt.Println(p.Inspect())          // 2026-06-29 05:18:32 +0530
}
```

## The clock seam

`Now()` is the only non-deterministic constructor. It reads the wall clock through
the package-level `NowFunc` seam, which a test pins to a fixed instant:

```go
old := rtime.NowFunc
rtime.NowFunc = func() (sec int64, nsec int32) { return 1782710312, 0 }
defer func() { rtime.NowFunc = old }()
// rtime.Now() is now exactly 2026-06-29 05:18:32 …
```

## API

```go
type Time struct{ /* (seconds, nanoseconds, utc-offset, is-utc) */ }

// Clock seam — the only non-determinism.
var NowFunc func() (sec int64, nsec int32)

// Constructors
func Now() *Time
func At(sec float64) *Time
func AtNsec(sec, nsec int64) *Time
func AtTime(t *Time) *Time
func UTC(year int, rest ...int) *Time   // Time.utc / gm
func Local(year int, rest ...int) *Time // Time.local / mktime
func New(year, month, day, hour, min, sec, offset int) *Time
func Parse(input string) (*Time, error)
func Strptime(input, layout string) (*Time, error)

// Components, conversions, zone shifts, arithmetic, ordering, formatting —
// see accessors.go, arith.go, and format.go.
func (t *Time) Year() int; func (t *Time) Month() int /* … */
func (t *Time) ToI() int64; func (t *Time) ToF() float64; func (t *Time) ToR() *big.Rat
func (t *Time) UTCTime() *Time; func (t *Time) GetLocal(offset int) *Time
func (t *Time) Add(secs float64) *Time; func (t *Time) Diff(other *Time) float64
func (t *Time) Cmp(other *Time) int; func (t *Time) Equal(other *Time) bool
func (t *Time) Strftime(format string) string
func (t *Time) Inspect() string; func (t *Time) ISO8601(fraction ...int) string
func (t *Time) RFC2822() string; func (t *Time) HTTPDate() string; func (t *Time) CTime() string
```

## Tests & coverage

The suite pairs deterministic, ruby-free golden tests (which alone hold coverage
at 100%, so the qemu cross-arch and Windows lanes pass the gate) with a
**differential MRI oracle**: every formatter, the full `strftime` directive
corpus, the numeric accessors, arithmetic, and `Parse` / `Strptime` are
cross-checked against the system `ruby`. The oracle scripts `$stdout.binmode` /
`$stdin.binmode` (so Windows text-mode never rewrites the bytes), run ruby with
`TZ=UTC`, gate themselves on `RUBY_VERSION >= "4.0"`, and skip where `ruby` is
absent. `TestMain` pins the process zone to UTC, so the deterministic tests are
reproducible on any runner regardless of its `TZ`.

```sh
COVERPKG=$(go list ./... | paste -sd, -)
go test -race -coverpkg="$COVERPKG" -coverprofile=cover.out ./...
go tool cover -func=cover.out | tail -1   # 100.0%
```

## License

BSD-3-Clause — see [LICENSE](LICENSE). Copyright the go-ruby-time/time authors.
