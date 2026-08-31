// Copyright (c) the go-macos authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

//go:build darwin

package multitouch

import (
	"sync/atomic"
	"testing"
	"time"
)

// TestTheLayoutIsRight is what makes the contact layout knowledge rather than
// folklore.
//
// MultitouchSupport publishes no header, so the struct in this package came
// from other people's readers of it. A wrong layout does not crash: it yields
// plausible-looking floats from the wrong offsets. The check that catches that
// is the framework's own contract -- normalised coordinates are in [0,1] -- so
// this reads real contacts and requires every one of them to be inside it.
//
// It NEVER prints a position. Counts and ranges say everything the test needs,
// and a trackpad trace is a record of somebody's hands.
//
// Nobody has to touch anything: the test passes on a machine with no fingers on
// the trackpad, having read no contacts, and says so. It is a check that the
// reading is RIGHT, not that it happened.
func TestTheLayoutIsRight(t *testing.T) {
	var inside, outside, frames int64
	w, err := Watch(func(f Frame) {
		atomic.AddInt64(&frames, 1)
		for _, c := range f.Contacts {
			if c.X < 0 || c.X > 1 || c.Y < 0 || c.Y > 1 {
				atomic.AddInt64(&outside, 1)
			} else {
				atomic.AddInt64(&inside, 1)
			}
		}
	})
	if err != nil {
		t.Skipf("no trackpad to read: %v", err)
	}
	defer w.Close()

	time.Sleep(2 * time.Second)
	in, out, n := atomic.LoadInt64(&inside), atomic.LoadInt64(&outside), atomic.LoadInt64(&frames)
	t.Logf("%d frame(s), %d contact(s) inside [0,1], %d outside", n, in, out)
	if out > 0 {
		t.Errorf("%d contact(s) outside [0,1]: the struct layout is wrong", out)
	}
	if n == 0 {
		t.Log("the device sent nothing in two seconds; on a Mac in use it sends about fifty frames a second")
	}
}

// TestCloseTwice: a watcher is stopped by whoever owns it and by a deferred
// call, and those are often the same watcher.
func TestCloseTwice(t *testing.T) {
	w, err := Watch(func(Frame) {})
	if err != nil {
		t.Skipf("no trackpad: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Errorf("first Close: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}
}

func TestWatchNeedsSomewhereToSend(t *testing.T) {
	if _, err := Watch(nil); err == nil {
		t.Error("Watch(nil) was accepted")
	}
}
