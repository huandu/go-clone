// Copyright 2023 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

package clone

import (
	"reflect"
	"runtime"
	"unsafe"
)

var typeOfAllocator = reflect.TypeOf(Allocator{})

// Allocator is a utility type for memory allocation.
type Allocator struct {
	pool      unsafe.Pointer
	new       func(pool unsafe.Pointer, t reflect.Type) reflect.Value
	makeSlice func(pool unsafe.Pointer, t reflect.Type, len, cap int) reflect.Value
	makeMap   func(pool unsafe.Pointer, t reflect.Type, n int) reflect.Value
	makeChan  func(pool unsafe.Pointer, t reflect.Type, buffer int) reflect.Value
}

// AllocatorMethods defines all methods required by allocator.
// If any of these methods is nil, allocator will use default method which allocates memory from heap.
type AllocatorMethods struct {
	New       func(pool unsafe.Pointer, t reflect.Type) reflect.Value
	MakeSlice func(pool unsafe.Pointer, t reflect.Type, len, cap int) reflect.Value
	MakeMap   func(pool unsafe.Pointer, t reflect.Type, n int) reflect.Value
	MakeChan  func(pool unsafe.Pointer, t reflect.Type, buffer int) reflect.Value
}

// FromHeap returns an allocator which allocate memory from heap.
func FromHeap() *Allocator {
	return heapAllocator
}

// NewAllocator creates an allocator which allocate memory from the pool.
//
// If methods.New is not nil, the allocator itself is created by calling methods.New.
func NewAllocator(pool unsafe.Pointer, methods *AllocatorMethods) (allocator *Allocator) {
	if methods.New == nil {
		allocator = &Allocator{
			pool:      pool,
			new:       methods.New,
			makeSlice: methods.MakeSlice,
			makeMap:   methods.MakeMap,
			makeChan:  methods.MakeChan,
		}
	} else {
		// Allocate the allocator from the pool.
		val := methods.New(pool, typeOfAllocator)
		allocator = (*Allocator)(unsafe.Pointer(val.Pointer()))
		runtime.KeepAlive(val)

		*allocator = Allocator{
			pool:      pool,
			new:       methods.New,
			makeSlice: methods.MakeSlice,
			makeMap:   methods.MakeMap,
			makeChan:  methods.MakeChan,
		}
	}

	if allocator.new == nil {
		allocator.new = heapNew
	}

	if allocator.makeSlice == nil {
		allocator.makeSlice = heapMakeSlice
	}

	if allocator.makeMap == nil {
		allocator.makeMap = heapMakeMap
	}

	if allocator.makeChan == nil {
		allocator.makeChan = heapMakeChan
	}

	return allocator
}

// New returns a new zero value of t.
func (a *Allocator) New(t reflect.Type) reflect.Value {
	return a.new(a.pool, t)
}

// MakeSlice creates a new zero-initialized slice value of t with len and cap.
func (a *Allocator) MakeSlice(t reflect.Type, len, cap int) reflect.Value {
	return a.makeSlice(a.pool, t, len, cap)
}

// MakeMap creates a new map with minimum size n.
func (a *Allocator) MakeMap(t reflect.Type, n int) reflect.Value {
	return a.makeMap(a.pool, t, n)
}

// MakeChan creates a new chan with buffer.
func (a *Allocator) MakeChan(t reflect.Type, buffer int) reflect.Value {
	return a.makeChan(a.pool, t, buffer)
}

// Clone recursively deep clone val to a new value with memory allocated from a.
func (a *Allocator) Clone(val reflect.Value) reflect.Value {
	return a.clone(val, true)
}

func (a *Allocator) clone(val reflect.Value, inCustomFunc bool) reflect.Value {
	if !val.IsValid() {
		return val
	}

	state := &cloneState{
		allocator: a,
	}

	if inCustomFunc {
		state.skipCustomFuncValue = val
	}

	return state.clone(val)
}

// CloneSlowly recursively deep clone val to a new value with memory allocated from a.
// It marks all cloned values internally, thus it can clone v with cycle pointer.
func (a *Allocator) CloneSlowly(val reflect.Value) reflect.Value {
	return a.cloneSlowly(val, true)
}

func (a *Allocator) cloneSlowly(val reflect.Value, inCustomFunc bool) reflect.Value {
	if !val.IsValid() {
		return val
	}

	state := &cloneState{
		allocator: a,
		visited:   visitMap{},
		invalid:   invalidPointers{},
	}

	if inCustomFunc {
		state.skipCustomFuncValue = val
	}

	cloned := state.clone(val)
	state.fix(cloned)
	return cloned
}

// The heapAllocator allocates memory from heap.
var heapAllocator = &Allocator{
	new:       heapNew,
	makeSlice: heapMakeSlice,
	makeMap:   heapMakeMap,
	makeChan:  heapMakeChan,
}

func heapNew(pool unsafe.Pointer, t reflect.Type) reflect.Value {
	return reflect.New(t)
}

func heapMakeSlice(pool unsafe.Pointer, t reflect.Type, len, cap int) reflect.Value {
	return reflect.MakeSlice(t, len, cap)
}

func heapMakeMap(pool unsafe.Pointer, t reflect.Type, n int) reflect.Value {
	return reflect.MakeMapWithSize(t, n)
}

func heapMakeChan(pool unsafe.Pointer, t reflect.Type, buffer int) reflect.Value {
	return reflect.MakeChan(t, buffer)
}
