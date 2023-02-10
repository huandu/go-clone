// Copyright 2023 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

package clone

import (
	"reflect"
	"testing"
	"unsafe"

	"github.com/huandu/go-assert"
)

func TestAllocatorClone(t *testing.T) {
	a := assert.New(t)
	cnt := 0
	allocator := NewAllocator(nil, &AllocatorMethods{
		New: func(pool unsafe.Pointer, t reflect.Type) reflect.Value {
			cnt++
			return heapNew(pool, t)
		},
	})

	type dataNode struct {
		Data int
		Next *dataNode
	}
	data := &dataNode{
		Data: 1,
		Next: &dataNode{
			Data: 2,
		},
	}
	cloned := allocator.Clone(reflect.ValueOf(data)).Interface().(*dataNode)
	a.Equal(data, cloned)

	// Should allocate following value.
	//     - allocator
	//     - data
	//     - data.Next
	a.Equal(cnt, 3)
}

func TestAllocatorCloneSlowly(t *testing.T) {
	a := assert.New(t)
	cnt := 0
	allocator := NewAllocator(nil, &AllocatorMethods{
		New: func(pool unsafe.Pointer, t reflect.Type) reflect.Value {
			cnt++
			return heapNew(pool, t)
		},
	})

	type dataNode struct {
		Data int
		Next *dataNode
	}

	// data is a cycle linked list.
	data := &dataNode{
		Data: 1,
		Next: &dataNode{
			Data: 2,
			Next: &dataNode{
				Data: 3,
			},
		},
	}
	data.Next.Next.Next = data

	cloned := allocator.CloneSlowly(reflect.ValueOf(data)).Interface().(*dataNode)

	a.Equal(data.Data, cloned.Data)
	a.Equal(data.Next.Data, cloned.Next.Data)
	a.Equal(data.Next.Next.Data, cloned.Next.Next.Data)
	a.Equal(data.Next.Next.Next.Data, cloned.Next.Next.Next.Data)
	a.Equal(data.Next.Next.Next.Next.Data, cloned.Next.Next.Next.Next.Data)
	a.Assert(cloned.Next.Next.Next == cloned)

	// Should allocate following value.
	//     - allocator
	//     - data
	//     - data.Next
	//     - data.Next.Next
	a.Equal(cnt, 4)
}
