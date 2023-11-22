// Copyright 2022 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

//go:build go1.19
// +build go1.19

package clone

import (
	"reflect"
	"sync/atomic"
)

// Record the count of cloning atomic.Pointer[T] for test purpose only.
var registerAtomicPointerCalled int32

// RegisterAtomicPointer registers a custom clone function for atomic.Pointer[T].
func RegisterAtomicPointer[T any]() {
	SetCustomFunc(reflect.TypeOf(atomic.Pointer[T]{}), func(allocator *Allocator, old, new reflect.Value) {
		if !old.CanAddr() {
			return
		}

		// Clone value inside atomic.Pointer[T].
		oldValue := old.Addr().Interface().(*atomic.Pointer[T])
		newValue := new.Addr().Interface().(*atomic.Pointer[T])
		v := oldValue.Load()
		newValue.Store(v)

		atomic.AddInt32(&registerAtomicPointerCalled, 1)
	})
}
