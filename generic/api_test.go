// Copyright 2022 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

package clone

import (
	"reflect"
	"testing"

	"github.com/huandu/go-assert"
)

type MyType struct {
	Foo int
	bar string
}

func TestGenericAPI(t *testing.T) {
	a := assert.New(t)
	original := &MyType{
		Foo: 123,
		bar: "player",
	}

	var v *MyType = Clone(original)
	a.Equal(v, original)

	v = Slowly(original)
	a.Equal(v, original)

	v = Wrap(original)
	a.Equal(v, original)
	a.Assert(Unwrap(v) == original)

	v.Foo = 777
	a.Equal(Unwrap(v).Foo, original.Foo)

	Undo(v)
	a.Equal(v, original)
}

type MyPointer struct {
	Foo *int
	P   *MyPointer
}

func TestMarkAsAPI(t *testing.T) {
	a := assert.New(t)
	MarkAsScalar(reflect.TypeOf(MyPointer{}))
	MarkAsOpaquePointer(reflect.TypeOf(&MyPointer{}))

	n := 0
	orignal := MyPointer{
		Foo: &n,
	}
	orignal.P = &orignal

	v := Clone(orignal)
	a.Assert(v.Foo == orignal.Foo)
	a.Assert(v.P == &orignal)
}
