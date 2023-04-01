// Copyright 2023 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

//go:build !goexperiment.arenas

package clone

import (
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"unsafe"
)

func ExampleAllocator() {
	// We can create a new allocator to hold customized config without poluting the default allocator.
	// Calling FromHeap() is a convenient way to create a new allocator which allocates memory from heap.
	allocator := FromHeap()

	// Mark T as scalar only in the allocator.
	type T struct {
		Value *int
	}
	allocator.MarkAsScalar(reflect.TypeOf(new(T)))

	t := &T{
		Value: new(int),
	}
	cloned1 := allocator.Clone(reflect.ValueOf(t)).Interface().(*T)
	cloned2 := Clone(t).(*T)

	fmt.Println(t.Value == cloned1.Value)
	fmt.Println(t.Value == cloned2.Value)

	// Output:
	// true
	// false
}

func ExampleAllocator_syncPool() {
	type Foo struct {
		Bar int
	}

	typeOfFoo := reflect.TypeOf(Foo{})
	poolUsed := 0 // For test only.

	// A sync pool to allocate Foo.
	p := &sync.Pool{
		New: func() interface{} {
			return &Foo{}
		},
	}

	// Creates a custom allocator using p as pool.
	allocator := NewAllocator(unsafe.Pointer(p), &AllocatorMethods{
		New: func(pool unsafe.Pointer, t reflect.Type) reflect.Value {
			// If t is Foo, allocate value from the sync pool p.
			if t == typeOfFoo {
				poolUsed++ // For test only.

				p := (*sync.Pool)(pool)
				v := p.Get()
				runtime.SetFinalizer(v, func(v *Foo) {
					*v = Foo{}
					p.Put(v)
				})

				return reflect.ValueOf(v)
			}

			// Fallback to reflect API.
			return reflect.New(t)
		},
	})

	// Do clone.
	target := []*Foo{
		{Bar: 1},
		{Bar: 2},
	}
	cloned := allocator.Clone(reflect.ValueOf(target)).Interface().([]*Foo)

	fmt.Println(reflect.DeepEqual(target, cloned))
	fmt.Println(poolUsed)

	// Output:
	// true
	// 2
}

func ExampleAllocator_deepCloneString() {
	// By default, string is considered as scalar and copied by value.
	// In some cases, we may need to clone string deeply, that is, copy the underlying bytes.
	// We can use a custom allocator to do this.
	allocator := NewAllocator(nil, &AllocatorMethods{
		IsScalar: func(t reflect.Kind) bool {
			return t != reflect.String && IsScalar(t)
		},
	})

	data := []byte("bytes")
	s1 := *(*string)(unsafe.Pointer(&data))             // Unsafe conversion from []byte to string.
	s2 := Clone(s1).(string)                            // s2 shares the same underlying bytes with s1.
	s3 := allocator.Clone(reflect.ValueOf(s1)).String() // s3 has its own underlying bytes.

	copy(data, "magic") // Change the underlying bytes.
	fmt.Println(s1)
	fmt.Println(s2)
	fmt.Println(s3)

	// Output:
	// magic
	// magic
	// bytes
}
