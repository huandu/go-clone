// Copyright 2023 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

package clone

import (
	"reflect"
	"sync"
	"testing"
	"unsafe"

	"github.com/huandu/go-assert"
)

func TestAllocatorMethodsParent(t *testing.T) {
	a := assert.New(t)
	parent := NewAllocator(nil, &AllocatorMethods{
		IsScalar: func(k reflect.Kind) bool {
			return k == reflect.Int
		},
	})
	allocator := NewAllocator(nil, &AllocatorMethods{
		Parent: parent,
	})

	a.Assert(parent.parent == defaultAllocator)
	a.Assert(allocator.parent == parent)

	// Set up customizations in parent.
	type T1 struct {
		Data []byte
	}
	type T2 struct {
		Data []byte
	}
	type T3 struct {
		Data []byte
	}
	typeOfT1 := reflect.TypeOf(new(T1))
	typeOfT2 := reflect.TypeOf(new(T2))
	typeOfT3 := reflect.TypeOf(new(T3))
	customFuncCalled := 0
	parent.MarkAsScalar(typeOfT1)
	parent.MarkAsOpaquePointer(typeOfT2)
	parent.SetCustomFunc(typeOfT3, func(allocator *Allocator, old, new reflect.Value) {
		customFuncCalled++
	})

	// All customizations should be inherited from parent.
	st1 := allocator.loadStructType(typeOfT1.Elem())
	st2 := allocator.loadStructType(typeOfT2.Elem())
	st3 := allocator.loadStructType(typeOfT3.Elem())
	a.Equal(len(st1.PointerFields), 0)
	a.Assert(st1.fn == nil)
	a.Equal(len(st3.PointerFields), 1)
	a.Assert(st2.fn == nil)
	a.Equal(len(st3.PointerFields), 1)
	a.Assert(st3.fn != nil)
	a.Assert(!allocator.isOpaquePointer(typeOfT1))
	a.Assert(allocator.isOpaquePointer(typeOfT2))
	a.Assert(!allocator.isOpaquePointer(typeOfT3))
	a.Assert(allocator.isScalar(reflect.Int))
	a.Assert(!allocator.isScalar(reflect.Uint))
}

func TestAllocatorMethodsPool(t *testing.T) {
	a := assert.New(t)
	pool1Called := 0
	pool1 := &sync.Pool{
		New: func() interface{} {
			pool1Called++
			return nil
		},
	}
	pool2Called := 0
	pool2 := &sync.Pool{
		New: func() interface{} {
			pool2Called++
			return nil
		},
	}
	parent := NewAllocator(unsafe.Pointer(pool1), &AllocatorMethods{
		New: func(pool unsafe.Pointer, t reflect.Type) reflect.Value {
			p := (*sync.Pool)(pool)
			p.Get()
			return defaultAllocator.New(t)
		},
		MakeSlice: func(pool unsafe.Pointer, t reflect.Type, len, cap int) reflect.Value {
			p := (*sync.Pool)(pool)
			p.Get()
			return defaultAllocator.MakeSlice(t, len, cap)
		},
		MakeMap: func(pool unsafe.Pointer, t reflect.Type, size int) reflect.Value {
			p := (*sync.Pool)(pool)
			p.Get()
			return defaultAllocator.MakeMap(t, size)
		},
	})
	allocator := NewAllocator(unsafe.Pointer(pool2), &AllocatorMethods{
		Parent: parent,
		MakeChan: func(pool unsafe.Pointer, t reflect.Type, size int) reflect.Value {
			p := (*sync.Pool)(pool)
			p.Get()
			return defaultAllocator.MakeChan(t, size)
		},
	})

	// All allocation should be implemented by parent.
	allocator.New(reflect.TypeOf(1))
	allocator.MakeSlice(reflect.TypeOf([]int{}), 0, 0)
	allocator.MakeMap(reflect.TypeOf(map[int]int{}), 0)
	allocator.MakeChan(reflect.TypeOf(make(chan int)), 0)

	// 1 for new parent allocator itself.
	// 1 for new allocator itself.
	// 3 for New, MakeSlice and MakeMap.
	a.Equal(pool1Called, 5)

	// 1 for MakeChan.
	a.Equal(pool2Called, 1)
}
