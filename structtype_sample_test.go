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

// ExampleSetCustomPtrFunc tests whether we can customize the cloning behavior of a pointer type.
// In this example, the custom function reuses cached values.
func ExampleSetCustomPtrFunc() {
	type Data struct {
		Name  string
	Value int
	}
	refs := make(map[string]*Data)
	// Filter nil values in Data when cloning old value.
	SetCustomPtrFunc(reflect.TypeOf(&Data{}), func(allocator *Allocator, old, new reflect.Value) {
		// The new is a zero value of MyStruct.
		// We can get its address to update it.
		value := new.Addr().Interface().(**Data)
		oldRole := old.Interface().(*Data)

		if cached, ok := refs[(*oldRole).Name]; ok {
			*value = cached
		} else {
			*value = &Data{
				Name:  (*oldRole).Name,
				Value: (*oldRole).Value,
			}
			refs[(*value).Name] = *value
		}
	})

	orig := &Data{
		Name:  "abc",
		Value: 123,
	}
	cloned1 := Clone(orig).(*Data)
	cloned2 := Clone(orig).(*Data)
	cloned1.Value = 456
	orig.Value = -1
	fmt.Println(*orig, *cloned1, *cloned2)

	// Output:
	// {abc -1} {abc 456} {abc 456}
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
