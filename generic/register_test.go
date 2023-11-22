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
	s := &Pointers{}
	stackPointerCannotBeCloned := atomic.Pointer[RegisteredPayload]{}

	// Register atomic.Pointer[RegisteredPayload] only.
	RegisterAtomicPointer[RegisteredPayload]()

	prev := registerAtomicPointerCalled
	Clone(s)
	Clone(stackPointerCannotBeCloned)
	a.Equal(registerAtomicPointerCalled, prev+1)
}
