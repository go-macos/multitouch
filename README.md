# multitouch

[![Go Reference](https://pkg.go.dev/badge/github.com/go-macos/multitouch.svg)](https://pkg.go.dev/github.com/go-macos/multitouch)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue.svg)](LICENSE)
[![Pure Go](https://img.shields.io/badge/pure%20Go-CGO%3D0-00ADD8?logo=go&logoColor=white)](https://github.com/go-macos/multitouch)

**The raw contacts on a Mac's trackpad, in pure Go, with no permission at all.**

The gestures macOS publishes are the ones macOS has already decided about. A
three-finger swipe means *switch space* only if that is switched on; it arrives
as a private event the Dock consumes first; and an application that wants the
gesture for something else has to fight for it — or ask a person to change a
system setting so that a program can work, which is backwards.

The contacts underneath are not spoken for.

```go
w, err := multitouch.Watch(func(f multitouch.Frame) {
    if sw, ok := swiper.Feed(f); ok && sw.Horizontal() {
        // three fingers went sideways
    }
})
defer w.Close()
```

- **No permission.** Not Accessibility, not Screen Recording, not an event tap.
- **Every device.** A Mac with a Magic Trackpad beside its built-in one has two,
  and listening to the wrong one hears nothing while somebody uses the other.
- **Normalised coordinates**, 0 to 1 across the surface — so a gesture on a
  laptop trackpad and the same one on a Magic Trackpad twice the size give the
  same numbers.
- Measured on an M4 Max: **2918 frames in sixty seconds**, one to four contacts
  distinguished.

## The recogniser is separate, and that is on purpose

`Swiper` turns frames into swipes: how many fingers, how far, within how long,
and once per gesture rather than once per frame past the threshold. It is
arithmetic over values, so it is tested against frames made up in a test file
rather than against somebody's hand — a recogniser that can only be tried by
swiping is a recogniser nobody changes. That half carries a 100% coverage gate.

## A private framework, and how that is handled

`MultitouchSupport` publishes no header. The contact layout here is the one
every open-source reader of it uses, and it is **checked rather than trusted**:
normalised coordinates are in `[0,1]` by definition, so a wrong layout shows up
as coordinates outside it instead of as plausible nonsense. `TestTheLayoutIsRight`
asserts exactly that against real contacts, and passes on a machine nobody is
touching — it checks that the reading is right, not that it happened.

## What this will not do

It reads where fingers are, because that is what a gesture is made of. It never
logs one. The tests print counts and ranges, never a position, and anything
built on this should keep that discipline: a trackpad trace is a record of
somebody's hands.
