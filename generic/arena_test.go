// Copyright 2023 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

//go:build go1.20 && goexperiment.arenas
// +build go1.20,goexperiment.arenas

package clone

import (
	"arena"
	"reflect"
	"runtime"
	"testing"
	"unsafe"

	"github.com/huandu/go-assert"
)

func TestArenaClone(t *testing.T) {
	a := assert.New(t)

	type FooInner struct {
		Value float64
	}

	type Foo struct {
		A string
		B []int
		C *FooInner
		D map[int]string
	}

	foo := &Foo{
		A: "hello",
		B: []int{1, 2, 3},
		C: &FooInner{
			Value: 45.6,
		},
		D: map[int]string{
			7: "7",
		},
	}

	ar := arena.NewArena()

	cloned := ArenaClone(ar, foo)
	a.Equal(foo, cloned)

	// If a pointer is not allocated by arena, arena.Clone() will return the pointer as it is.
	// Use this feature to check whether a pointer is allocated by arena.
	prevStr := foo.A
	str := arena.Clone(cloned.A)
	a.Assert(((*reflect.StringHeader)(unsafe.Pointer(&str))).Data != ((*reflect.StringHeader)(unsafe.Pointer(&prevStr))).Data)

	a.Assert(arena.Clone(cloned) != foo)

	slice := arena.Clone(cloned.B)
	a.Assert(&slice[0] != &foo.B[0])

	a.Assert(arena.Clone(cloned.C) != foo.C)

	prevStr = foo.D[7]
	str = arena.Clone(cloned.D[7])
	a.Assert(((*reflect.StringHeader)(unsafe.Pointer(&str))).Data != ((*reflect.StringHeader)(unsafe.Pointer(&prevStr))).Data)

	// Make sure ar is alive.
	runtime.KeepAlive(ar)
}
