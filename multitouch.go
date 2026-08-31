// Copyright (c) the go-macos authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

// Package multitouch reads the raw contacts on a Mac's trackpad.
//
// It exists because the gestures macOS publishes are the ones macOS has already
// decided about. A three-finger swipe means "switch space" only if that is
// switched on, it is delivered as a private event the Dock consumes first, and
// an application that wants the gesture for something else has to fight for it
// -- or ask a person to change a system setting so that a program can work,
// which is backwards.
//
// The contacts underneath are not spoken for. Reading them needs NO permission:
// not Accessibility, not Screen Recording, not an event tap. Measured on an
// M4 Max: 2918 frames in sixty seconds, one to four contacts distinguished.
//
// The consumer it was written for is a virtual desktop worn on the face, where
// three fingers sliding sideways should turn the ribbon of screens -- something
// no system gesture is going to offer, on a machine where the same swipe is
// bound to nothing.
//
// # A private framework
//
// MultitouchSupport publishes no header. The contact layout here is the one
// every open-source reader of it uses, and it is CHECKED rather than trusted:
// normalised coordinates are in [0,1] by definition, so a wrong layout shows up
// as coordinates outside it rather than as plausible nonsense. See
// TestTheLayoutIsRight, which asserts that on real contacts.
//
// # What this package will not do
//
// It reads where fingers are, because that is what a gesture is made of. It
// never logs one. A test here prints counts and ranges, never a position, and
// anything built on it should keep that discipline: a trackpad trace is a
// record of somebody's hands.
package multitouch

import (
	"errors"
	"time"
)

// ErrUnsupported is returned on platforms with no multitouch device to read.
var ErrUnsupported = errors.New("multitouch: only macOS has this")

// Contact is one finger on the surface.
//
// X and Y are NORMALISED: 0 to 1 across the trackpad, origin at the bottom
// left, which is the framework's own convention and the reason a caller can
// compare a gesture on a laptop's trackpad with the same gesture on a Magic
// Trackpad twice the size.
type Contact struct {
	// ID follows one finger for as long as it stays down, so a caller can tell
	// three fingers moving together from three that landed and lifted.
	ID int
	// X and Y are 0..1 across the surface.
	X, Y float32
	// VX and VY are the same units per second.
	VX, VY float32
	// Size is how much of the surface the contact covers, in the framework's
	// own arbitrary units: useful for telling a finger from a resting palm,
	// meaningless as an absolute number.
	Size float32
}

// Frame is every contact on one device at one instant.
type Frame struct {
	// At is the device's own timestamp, which is monotonic and NOT wall clock:
	// use it for durations between frames, never for what time something
	// happened.
	At time.Duration
	// Device tells frames from two trackpads apart. A Mac with a Magic
	// Trackpad beside its built-in one has two, and a caller that mixes them
	// sees six fingers where there are three.
	Device   int
	Contacts []Contact
}
