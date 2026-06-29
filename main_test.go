// Copyright (c) the go-ruby-time/time authors
//
// SPDX-License-Identifier: BSD-3-Clause

package time

import (
	"os"
	"testing"
	stdtime "time"
)

// TestMain pins the process-local zone to UTC for the whole suite so every
// zone-dependent expectation (a zone-less Strptime/Parse, Time#localtime, the
// Local constructor) is deterministic on any CI machine — independent of the
// runner's TZ. The Windows / qemu / no-ruby lanes that do not export TZ=UTC then
// still produce the golden strings, keeping the 100% gate machine-independent.
func TestMain(m *testing.M) {
	stdtime.Local = stdtime.UTC
	os.Exit(m.Run())
}
