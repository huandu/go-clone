// Copyright 2023 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

package clone

import (
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"unsafe"
)

func ExampleAllocator() {
	type Foo struct {
		Bar int
	}
	var typeOfFoo = reflect.TypeOf(Foo{})

	// A sync pool to allocate Foo.
	p := &sync.Pool{
		New: func() interface{} {
			return &Foo{}
		},
	}
	counter := 0

	// Creates a custom allocator using p as pool.
	allocator := NewAllocator(unsafe.Pointer(p), &AllocatorMethods{
		New: func(pool unsafe.Pointer, t reflect.Type) reflect.Value {
			// If t is Foo, allocate value from the sync pool p.
			if t == typeOfFoo {
				p := (*sync.Pool)(pool)
				v := p.Get()
				runtime.SetFinalizer(v, func(v *Foo) {
					*v = Foo{}
					p.Put(v)
				})

				// Add counter for test. So we know whether this code was executed.
				counter++

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
	fmt.Println(counter)

	// Output:
	// true
	// 2
}
