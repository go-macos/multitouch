// Copyright (c) the go-macos authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

//go:build !darwin

package multitouch

// Watcher is the same type everywhere, so a caller compiles unchanged.
type Watcher struct{}

// Watch reports ErrUnsupported: the framework it reads is macOS's.
func Watch(func(Frame)) (*Watcher, error) { return nil, ErrUnsupported }

// Close does nothing where nothing was started.
func (w *Watcher) Close() error { return nil }
