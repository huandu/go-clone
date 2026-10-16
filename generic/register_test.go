// Copyright 2022 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

//go:build go1.19
// +build go1.19

package clone

import (
	"sync/atomic"
	"testing"

	"github.com/huandu/go-assert"
)

type RegisteredPayload struct {
	T string
}

type UnregisteredPayload struct {
	T string
}

type Pointers struct {
	P1 atomic.Pointer[RegisteredPayload]
	P2 atomic.Pointer[UnregisteredPayload]
}

func TestRegisterAtomicPointer(t *testing.T) {
	a := assert.New(t)

	// s has one registered (P1) and one unregistered (P2) atomic pointer field.
	s := &Pointers{}

	// Register atomic.Pointer[RegisteredPayload] only.
	RegisterAtomicPointer[RegisteredPayload]()

	prev := registerAtomicPointerCalled
	Clone(s)

	// Only P1 is registered, so exactly one field must have gone through the
	// custom clone func even though the struct has two atomic pointer fields.
	a.Equal(registerAtomicPointerCalled, prev+1)

	prev = registerAtomicPointerCalled

	// The custom clone func is only reachable for addressable values (it bails out
	// when old.CanAddr() is false), and a non-addressable value has no addressable
	// field either. Passing a freshly built atomic.Pointer value covers that path
	// and must not invoke the callback.
	//
	// This deliberately copies an atomic.Pointer, which go vet reports as
	// copylocks when the argument is a named variable. The copy is safe here: the
	// value is a fresh zero value that has never been stored to or shared with
	// another goroutine, and Clone only reads it - the copy exists precisely to
	// observe what the clone path does with a non-addressable atomic value. A named
	// variable would make the value addressable and would no longer exercise this
	// path, so the composite literal is deliberate rather than incidental.
	Clone(atomic.Pointer[RegisteredPayload]{})

	a.Equal(registerAtomicPointerCalled, prev)
}
