// Copyright (c) the go-macos authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

package multitouch

import "time"

// Defaults for a [Swiper]. They are exported because a caller tuning one wants
// to say what it changed FROM.
const (
	// DefaultFingers is three: two is a scroll everywhere on this platform, and
	// four is the system's own.
	DefaultFingers = 3
	// DefaultDistance is a seventh of the trackpad. Short enough to be one
	// comfortable motion, long enough that resting three fingers and shifting
	// slightly is not a gesture.
	DefaultDistance = 0.15
	// DefaultWithin bounds how long the fingers may take. A slow drift across
	// the surface is somebody resting their hand, not a swipe.
	DefaultWithin = 700 * time.Millisecond
)

// A Swipe is fingers that went somewhere together.
//
// Dx and Dy are the movement in the same normalised units as [Contact], so a
// swipe on a small trackpad and the same one on a large Magic Trackpad give the
// same numbers. Positive Dx is rightward, positive Dy is upward -- the
// framework's own axes, not a screen's.
type Swipe struct {
	Fingers int
	Dx, Dy  float32
}

// Horizontal reports whether this went sideways rather than up or down.
func (s Swipe) Horizontal() bool { return abs(s.Dx) > abs(s.Dy) }

// Right reports the direction of a horizontal swipe.
func (s Swipe) Right() bool { return s.Dx > 0 }

// Swiper turns frames into swipes.
//
// It is deliberately separate from the reading: this half is arithmetic over
// values, so it is tested against frames made up in a test file rather than
// against somebody's hand. A recogniser that can only be tried by swiping is a
// recogniser nobody changes.
//
// The zero value works and uses the defaults above.
type Swiper struct {
	// Fingers is how many contacts must move together; 0 means
	// [DefaultFingers].
	Fingers int
	// Distance is how far they must go, normalised; 0 means [DefaultDistance].
	Distance float32
	// Within bounds the gesture's duration; 0 means [DefaultWithin].
	Within time.Duration

	started bool
	startAt time.Duration
	startX  float32
	startY  float32
	// spent says the gesture has been reported and the fingers have not lifted
	// yet. Without it one long swipe fires again on every frame after the
	// threshold, which reads as a person swiping ten times.
	spent bool
}

// Feed offers one frame and reports a swipe when the fingers have completed
// one.
//
// It reports at most once per gesture: after a swipe, the fingers must LEAVE
// the surface before another can begin. That is not a nicety -- without it a
// single motion crosses the threshold on one frame and stays across it for
// every frame after, and a desk that changes screen per report would fly past
// six of them.
func (s *Swiper) Feed(f Frame) (Swipe, bool) {
	want := s.Fingers
	if want <= 0 {
		want = DefaultFingers
	}
	n := len(f.Contacts)

	// Nothing down: whatever was happening is over, including a spent gesture.
	if n == 0 {
		s.started, s.spent = false, false
		return Swipe{}, false
	}
	if n != want {
		// The wrong number of fingers is not this gesture. A fourth finger
		// landing mid-swipe ends it rather than changing it, because the person
		// is now doing something else.
		s.started = false
		return Swipe{}, false
	}
	x, y := centre(f.Contacts)
	if !s.started {
		s.started, s.startAt, s.startX, s.startY = true, f.At, x, y
		return Swipe{}, false
	}
	if s.spent {
		return Swipe{}, false
	}

	within := s.Within
	if within <= 0 {
		within = DefaultWithin
	}
	if f.At-s.startAt > within {
		// Too slow to be a swipe. Start again from here rather than refusing
		// until the fingers lift: a hand that rested and then moved is a
		// gesture, and the resting is not part of it.
		s.startAt, s.startX, s.startY = f.At, x, y
		return Swipe{}, false
	}

	dx, dy := x-s.startX, y-s.startY
	distance := s.Distance
	if distance <= 0 {
		distance = DefaultDistance
	}
	if abs(dx) < distance && abs(dy) < distance {
		return Swipe{}, false
	}
	s.spent = true
	return Swipe{Fingers: n, Dx: dx, Dy: dy}, true
}

// centre is where the fingers are, taken together. A swipe is the hand moving,
// and one finger sliding while the others stay is not one -- averaging is what
// makes the difference show up as a small number.
func centre(cs []Contact) (x, y float32) {
	for _, c := range cs {
		x, y = x+c.X, y+c.Y
	}
	n := float32(len(cs))
	return x / n, y / n
}

func abs(f float32) float32 {
	if f < 0 {
		return -f
	}
	return f
}
