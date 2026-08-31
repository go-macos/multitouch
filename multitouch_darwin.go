// Copyright (c) the go-macos authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

//go:build darwin

package multitouch

import (
	"fmt"
	"sync"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
)

const (
	multitouchFramework = "/System/Library/PrivateFrameworks/MultitouchSupport.framework/MultitouchSupport"
	coreFoundation      = "/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation"
)

// mtTouch is one contact as the framework lays it out.
//
// MultitouchSupport publishes no header, so this is the layout every
// open-source reader of it uses. It is checked rather than trusted: normalised
// coordinates are in [0,1] by definition, and a wrong layout puts them outside
// it. TestTheLayoutIsRight asserts that against real contacts.
type mtTouch struct {
	Frame      int32
	_          [4]byte // the timestamp that follows is 8-aligned
	Timestamp  float64
	Identifier int32
	State      int32
	_          int32
	_          int32
	NormPosX   float32
	NormPosY   float32
	NormVelX   float32
	NormVelY   float32
	Size       float32
	_          int32
	Angle      float32
	MajorAxis  float32
	MinorAxis  float32
	MmPosX     float32
	MmPosY     float32
	MmVelX     float32
	MmVelY     float32
	_          [2]int32
	_          float32
}

// The framework, loaded once. The seams are package variables so a test can
// drive the failures without a trackpad.
var (
	loadOnce sync.Once
	loadErr  error

	mtCreateList  func() uintptr
	mtCreateFirst func() uintptr
	mtRegister    func(dev uintptr, cb uintptr) int32
	mtStart       func(dev uintptr, mode int32) int32
	mtStop        func(dev uintptr) int32
	cfCount       func(a uintptr) int64
	cfAt          func(a uintptr, i int64) uintptr

	dlopen = func(path string) (uintptr, error) {
		return purego.Dlopen(path, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	}
)

func load() error {
	loadOnce.Do(func() {
		mt, err := dlopen(multitouchFramework)
		if err != nil {
			loadErr = fmt.Errorf("multitouch: %w", err)
			return
		}
		cf, err := dlopen(coreFoundation)
		if err != nil {
			loadErr = fmt.Errorf("multitouch: %w", err)
			return
		}
		purego.RegisterLibFunc(&mtCreateList, mt, "MTDeviceCreateList")
		purego.RegisterLibFunc(&mtCreateFirst, mt, "MTDeviceCreateDefault")
		purego.RegisterLibFunc(&mtRegister, mt, "MTRegisterContactFrameCallback")
		purego.RegisterLibFunc(&mtStart, mt, "MTDeviceStart")
		purego.RegisterLibFunc(&mtStop, mt, "MTDeviceStop")
		purego.RegisterLibFunc(&cfCount, cf, "CFArrayGetCount")
		purego.RegisterLibFunc(&cfAt, cf, "CFArrayGetValueAtIndex")
	})
	return loadErr
}

// Watcher is a subscription to the trackpads' contacts.
type Watcher struct {
	mu      sync.Mutex
	devices []uintptr
	stopped bool
}

// Watch calls fn for every frame from EVERY multitouch device, and keeps
// calling it until the watcher is closed.
//
// Every device, not the first: a Mac with a Magic Trackpad beside its built-in
// one has two, and listening to the wrong one hears nothing at all while
// somebody uses the other. The Frame says which device it came from.
//
// fn runs on the framework's own thread, once per frame -- about fifty to
// seventy times a second per device. It must not block: what it should do is
// hand the frame on and return.
func Watch(fn func(Frame)) (*Watcher, error) {
	if fn == nil {
		return nil, fmt.Errorf("multitouch: Watch needs a function to call")
	}
	if err := load(); err != nil {
		return nil, err
	}
	devices := allDevices()
	if len(devices) == 0 {
		return nil, fmt.Errorf("multitouch: no trackpad on this machine")
	}

	cb := purego.NewCallback(func(device uintptr, data unsafe.Pointer, n uintptr, timestamp float64, frame uintptr) uintptr {
		fn(Frame{
			At:       time.Duration(timestamp * float64(time.Second)),
			Device:   int(device),
			Contacts: contacts(data, int(n)),
		})
		return 0
	})

	w := &Watcher{}
	for _, d := range devices {
		mtRegister(d, cb)
		if rc := mtStart(d, 0); rc != 0 {
			// One device refusing is not the end: the other may be the one
			// under somebody's hand.
			continue
		}
		w.devices = append(w.devices, d)
	}
	if len(w.devices) == 0 {
		return nil, fmt.Errorf("multitouch: no device would start")
	}
	return w, nil
}

// contacts copies the framework's array into Go values.
//
// Copied rather than pointed at: the array belongs to the callback and is gone
// when it returns, so a Frame that kept a pointer would be a Frame that reads
// somebody else's memory a moment later.
// The array arrives as a POINTER rather than a uintptr, and stays one: go vet
// rejects turning an integer back into a pointer, and it is right to -- an
// address held as a number is an address the collector may move out from under.
func contacts(data unsafe.Pointer, n int) []Contact {
	if data == nil || n <= 0 {
		return nil
	}
	out := make([]Contact, 0, n)
	for i := range n {
		t := *(*mtTouch)(unsafe.Add(data, uintptr(i)*unsafe.Sizeof(mtTouch{})))
		out = append(out, Contact{
			ID:   int(t.Identifier),
			X:    t.NormPosX,
			Y:    t.NormPosY,
			VX:   t.NormVelX,
			VY:   t.NormVelY,
			Size: t.Size,
		})
	}
	return out
}

// allDevices lists every multitouch device, falling back to the default one.
func allDevices() []uintptr {
	var out []uintptr
	if list := mtCreateList(); list != 0 {
		for i := int64(0); i < cfCount(list); i++ {
			if d := cfAt(list, i); d != 0 {
				out = append(out, d)
			}
		}
	}
	if len(out) == 0 {
		if d := mtCreateFirst(); d != 0 {
			out = append(out, d)
		}
	}
	return out
}

// Close stops listening. It is safe to call twice.
func (w *Watcher) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.stopped {
		return nil
	}
	w.stopped = true
	for _, d := range w.devices {
		mtStop(d)
	}
	return nil
}
