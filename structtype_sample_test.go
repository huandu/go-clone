// Copyright 2019 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

package clone

import (
	"fmt"
	"os"
	"reflect"
)

func ExampleMarkAsScalar() {
	type ScalarType struct {
		stderr *os.File
	}

	MarkAsScalar(reflect.TypeOf(new(ScalarType)))

	scalar := &ScalarType{
		stderr: os.Stderr,
	}
	cloned := Clone(scalar).(*ScalarType)

	// cloned is a shadow copy of scalar
	// so that the pointer value should be the same.
	fmt.Println(scalar.stderr == cloned.stderr)

	// Output:
	// true
}

func ExampleMarkAsOpaquePointer() {
	type OpaquePointerType struct {
		foo int
	}

	MarkAsOpaquePointer(reflect.TypeOf(new(OpaquePointerType)))

	opaque := &OpaquePointerType{
		foo: 123,
	}
	cloned := Clone(opaque).(*OpaquePointerType)

	// cloned is a shadow copy of opaque.
	// so that opaque and cloned should be the same.
	fmt.Println(opaque == cloned)

	// Output:
	// true
}

func ExampleSetCustomFunc() {
	type MyStruct struct {
		Data []interface{}
	}

	// Filter nil values in Data when cloning old value.
	SetCustomFunc(reflect.TypeOf(MyStruct{}), func(allocator *Allocator, old, new reflect.Value) {
		// The new is a zero value of MyStruct.
		// We can get its address to update it.
		value := new.Addr().Interface().(*MyStruct)

		// The old is guaranteed to be a MyStruct.
		// As old.CanAddr() may be false, we'd better to read Data field directly.
		data := old.FieldByName("Data")
		l := data.Len()

		for i := 0; i < l; i++ {
			val := data.Index(i)

			if val.IsNil() {
				continue
			}

			n := allocator.Clone(val).Interface()
			value.Data = append(value.Data, n)
		}
	})

	slice := &MyStruct{
		Data: []interface{}{
			"abc", nil, 123, nil,
		},
	}
	cloned := Clone(slice).(*MyStruct)
	fmt.Println(cloned.Data)

	// Output:
	// [abc 123]
}
