// Copyright 2023 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

//go:build go1.20 && goexperiment.arenas
// +build go1.20,goexperiment.arenas

package clone

import (
	"arena"
	"reflect"
	"runtime"
	"unsafe"

	"github.com/huandu/go-clone"
)

// The arenaAllocator allocates memory from arena.
var arenaAllocatorMethods = &clone.AllocatorMethods{
	New:       arenaNew,
	MakeSlice: arenaMakeSlice,
	MakeMap:   arenaMakeMap,
	MakeChan:  arenaMakeChan,
}

// FromArena creates an allocator using arena a to allocate memory.
func FromArena(a *arena.Arena) *clone.Allocator {
	return clone.NewAllocator(unsafe.Pointer(a), arenaAllocatorMethods)
}

// ArenaClone recursively deep clones v to a new value in arena a.
// It works in the same way as Clone, except it allocates all memory from arena.
func ArenaClone[T any](a *arena.Arena, v T) (nv T) {
	src := reflect.ValueOf(v)
	cloned := FromArena(a).Clone(src)

	if !cloned.IsValid() {
		return
	}

	dst := reflect.ValueOf(&nv).Elem()
	dst.Set(cloned)
	return
}

// ArenaCloneSlowly recursively deep clones v to a new value in arena a.
// It works in the same way as Slowly, except it allocates all memory from arena.
func ArenaCloneSlowly[T any](a *arena.Arena, v T) (nv T) {
	src := reflect.ValueOf(v)
	cloned := FromArena(a).CloneSlowly(src)

	if !cloned.IsValid() {
		return
	}

	dst := reflect.ValueOf(&nv).Elem()
	dst.Set(cloned)
	return
}

func arenaNew(pool unsafe.Pointer, t reflect.Type) reflect.Value {
	return reflect.ArenaNew((*arena.Arena)(pool), reflect.PtrTo(t))
}

// Define the slice header again to mute golint's warning.
type sliceHeader reflect.SliceHeader

func arenaMakeSlice(pool unsafe.Pointer, t reflect.Type, len, cap int) reflect.Value {
	a := (*arena.Arena)(pool)

	// As of go1.20, there is no reflect method to allocate slice in arena.
	// Following code is a hack to allocate a large enough byte buffer
	// and then cast it to T[].
	et := t.Elem()
	l := int(et.Size())
	total := l * cap

	data := arena.MakeSlice[byte](a, total, total)
	ptr := unsafe.Pointer(&data[0])
	elem := reflect.NewAt(et, ptr)
	slicePtr := reflect.ArenaNew(a, reflect.PtrTo(t))
	*(*sliceHeader)(slicePtr.UnsafePointer()) = sliceHeader{
		Data: elem.Pointer(),
		Len:  l,
		Cap:  cap,
	}
	runtime.KeepAlive(elem)

	slice := slicePtr.Elem()
	return slice.Slice3(0, len, cap)
}

func arenaMakeMap(pool unsafe.Pointer, t reflect.Type, n int) reflect.Value {
	// As of go1.20, there is no way to allocate map in arena.
	// Fallback to heap allocation.
	return reflect.MakeMapWithSize(t, n)
}

func arenaMakeChan(pool unsafe.Pointer, t reflect.Type, buffer int) reflect.Value {
	// As of go1.20, there is no way to allocate chan in arena.
	// Fallback to heap allocation.
	return reflect.MakeChan(t, buffer)
}
