// Copyright 2019 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

package clone

import (
	"encoding/json"
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

func ExampleSetCustomFunc_partiallyClone() {
	type T struct {
		Value int
	}

	type MyStruct struct {
		S1 *T
		S2 string
		S3 int
	}

	SetCustomFunc(reflect.TypeOf(T{}), func(allocator *Allocator, old, new reflect.Value) {
		oldField := old.FieldByName("Value")
		newField := new.FieldByName("Value")
		newField.SetInt(oldField.Int() + 100)
	})

	SetCustomFunc(reflect.TypeOf(MyStruct{}), func(allocator *Allocator, old, new reflect.Value) {
		// We can call allocator.Clone to clone the old value without worrying about dead loop.
		// This custom func is temporary disabled for the old value in allocator.
		new.Set(allocator.Clone(old))

		oldField := old.FieldByName("S2")
		newField := new.FieldByName("S2")
		newField.SetString(oldField.String() + "_suffix")
	})

	st := &MyStruct{
		S1: &T{
			Value: 1,
		},
		S2: "abc",
		S3: 2,
	}
	cloned := Clone(st).(*MyStruct)

	data, _ := json.Marshal(st)
	fmt.Println(string(data))
	data, _ = json.Marshal(cloned)
	fmt.Println(string(data))

	// Output:
	// {"S1":{"Value":1},"S2":"abc","S3":2}
	// {"S1":{"Value":101},"S2":"abc_suffix","S3":2}
}

func ExampleSetCustomFunc_conditionalClonePointer() {
	type T struct {
		shouldClone bool
		data        []string
	}

	type Pointer struct {
		*T
	}

	values := map[string]Pointer{
		"shouldClone": {
			T: &T{
				shouldClone: true,
				data:        []string{"a", "b", "c"},
			},
		},
		"shouldNotClone": {
			T: &T{
				shouldClone: false,
				data:        []string{"a", "b", "c"},
			},
		},
	}
	SetCustomFunc(reflect.TypeOf(Pointer{}), func(allocator *Allocator, old, new reflect.Value) {
		p := old.Interface().(Pointer)

		if p.shouldClone {
			np := allocator.Clone(old).Interface().(Pointer)

			// Update the cloned value to make the change very obvious.
			np.shouldClone = false
			np.data = append(np.data, "cloned")
			new.Set(reflect.ValueOf(np))
		} else {
			new.Set(old)
		}
	})

	cloned := Clone(values).(map[string]Pointer)
	fmt.Println(cloned["shouldClone"].data)
	fmt.Println(cloned["shouldNotClone"].data)

	// Output:
	// [a b c cloned]
	// [a b c]
}
