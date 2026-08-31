// Copyright (c) the go-macos authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

package multitouch

import (
	"testing"
	"time"
)

// hand makes a frame with n fingers whose centre is at x,y. The fingers are
// spread a little, because a real hand is not a point and the recogniser must
// average rather than follow one contact.
func hand(at time.Duration, n int, x, y float32) Frame {
	f := Frame{At: at}
	for i := range n {
		off := float32(i-1) * 0.03
		f.Contacts = append(f.Contacts, Contact{ID: i, X: x + off, Y: y + off*0.1})
	}
	return f
}

// slide feeds a swiper a gesture of n fingers moving by dx,dy over steps
// frames, and reports what it recognised.
func slide(s *Swiper, n int, dx, dy float32, steps int, over time.Duration) []Swipe {
	var out []Swipe
	for i := 0; i <= steps; i++ {
		p := float32(i) / float32(steps)
		at := time.Duration(float64(over) * float64(p))
		if sw, ok := s.Feed(hand(at, n, 0.4+dx*p, 0.5+dy*p)); ok {
			out = append(out, sw)
		}
	}
	return out
}

func TestThreeFingersGoingRight(t *testing.T) {
	var s Swiper
	got := slide(&s, 3, 0.3, 0, 10, 300*time.Millisecond)
	if len(got) != 1 {
		t.Fatalf("recognised %d swipe(s), want exactly one: %v", len(got), got)
	}
	sw := got[0]
	if sw.Fingers != 3 {
		t.Errorf("Fingers = %d, want 3", sw.Fingers)
	}
	if !sw.Horizontal() {
		t.Errorf("Dx=%v Dy=%v was not called horizontal", sw.Dx, sw.Dy)
	}
	if !sw.Right() {
		t.Error("a swipe to the right was not called rightward")
	}
}

func TestThreeFingersGoingLeft(t *testing.T) {
	var s Swiper
	got := slide(&s, 3, -0.3, 0, 10, 300*time.Millisecond)
	if len(got) != 1 || got[0].Right() {
		t.Fatalf("got %v, want one leftward swipe", got)
	}
}

// TestOneGestureIsOneSwipe is the reason `spent` exists. Without it the
// threshold stays crossed for every frame after it is crossed, and a desk that
// changes screen per report flies past six of them on one motion.
func TestOneGestureIsOneSwipe(t *testing.T) {
	var s Swiper
	// Forty frames, far past the threshold, then the fingers stay down.
	if got := slide(&s, 3, 0.9, 0, 40, 400*time.Millisecond); len(got) != 1 {
		t.Errorf("one long motion gave %d swipes, want 1", len(got))
	}
	// Still down and still moving: nothing more.
	if _, ok := s.Feed(hand(500*time.Millisecond, 3, 0.9, 0.5)); ok {
		t.Error("the same gesture reported twice")
	}
	// Lifted, then a fresh swipe: that one counts.
	s.Feed(Frame{At: 600 * time.Millisecond})
	if got := slide(&s, 3, 0.3, 0, 10, 300*time.Millisecond); len(got) != 1 {
		t.Errorf("the next gesture gave %d swipes, want 1", len(got))
	}
}

func TestTheWrongNumberOfFingers(t *testing.T) {
	for _, n := range []int{1, 2, 4, 5} {
		var s Swiper
		if got := slide(&s, n, 0.4, 0, 10, 300*time.Millisecond); len(got) != 0 {
			t.Errorf("%d fingers were taken for the gesture: %v", n, got)
		}
	}
}

// TestAFingerLandingMidSwipeEndsIt: the person is now doing something else, and
// finishing their earlier motion for them would be a screen change they did not
// ask for.
func TestAFingerLandingMidSwipeEndsIt(t *testing.T) {
	var s Swiper
	s.Feed(hand(0, 3, 0.3, 0.5))
	s.Feed(hand(50*time.Millisecond, 3, 0.38, 0.5))
	if _, ok := s.Feed(hand(80*time.Millisecond, 4, 0.5, 0.5)); ok {
		t.Error("a four-finger frame completed a three-finger swipe")
	}
	// And the three-finger gesture has to start over rather than resume.
	if _, ok := s.Feed(hand(120*time.Millisecond, 3, 0.6, 0.5)); ok {
		t.Error("the interrupted gesture resumed and fired")
	}
}

// TestASlowDriftIsNotASwipe: a hand resting on the trackpad wanders. Given long
// enough, any wander crosses any distance.
func TestASlowDriftIsNotASwipe(t *testing.T) {
	var s Swiper
	if got := slide(&s, 3, 0.5, 0, 20, 5*time.Second); len(got) != 0 {
		t.Errorf("a five-second drift was called a swipe: %v", got)
	}
	// But a real swipe AFTER the drift still counts: the resting is not part of
	// the gesture, so the recogniser restarts rather than sulking.
	base := 5 * time.Second
	for i := 0; i <= 10; i++ {
		p := float32(i) / 10
		if sw, ok := s.Feed(hand(base+time.Duration(p*float32(200*time.Millisecond)), 3, 0.2+0.3*p, 0.5)); ok {
			if !sw.Horizontal() {
				t.Errorf("the swipe after the drift came out %v", sw)
			}
			return
		}
	}
	t.Error("a swipe after a drift was never recognised")
}

func TestVerticalIsNotHorizontal(t *testing.T) {
	var s Swiper
	got := slide(&s, 3, 0, 0.3, 10, 300*time.Millisecond)
	if len(got) != 1 {
		t.Fatalf("recognised %d, want one", len(got))
	}
	if got[0].Horizontal() {
		t.Errorf("Dx=%v Dy=%v was called horizontal", got[0].Dx, got[0].Dy)
	}
}

// TestOneFingerSlidingIsNotAHand: the centre is what moves in a swipe. One
// finger travelling while the other two stay put moves the centre by a third,
// which is exactly what averaging is for.
func TestOneFingerSlidingIsNotAHand(t *testing.T) {
	var s Swiper
	s.Feed(Frame{At: 0, Contacts: []Contact{{ID: 0, X: 0.3}, {ID: 1, X: 0.4}, {ID: 2, X: 0.5}}})
	_, ok := s.Feed(Frame{At: 100 * time.Millisecond, Contacts: []Contact{
		{ID: 0, X: 0.3}, {ID: 1, X: 0.4}, {ID: 2, X: 0.75}, // one finger moved 0.25
	}})
	if ok {
		t.Error("one finger moving was taken for three")
	}
}

func TestTheSettingsAreTheCallersToChange(t *testing.T) {
	s := Swiper{Fingers: 2, Distance: 0.05, Within: 2 * time.Second}
	if got := slide(&s, 2, 0.08, 0, 10, time.Second); len(got) != 1 {
		t.Errorf("a tuned swiper recognised %d, want 1", len(got))
	}
}

// TestNothingDownEndsEverything covers the frame with no contacts, which is
// what lifting a hand looks like and what resets a spent gesture.
func TestNothingDownEndsEverything(t *testing.T) {
	var s Swiper
	if _, ok := s.Feed(Frame{}); ok {
		t.Error("an empty frame reported a swipe")
	}
}
