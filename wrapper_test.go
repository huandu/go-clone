// Copyright 2019 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

package clone

import (
	"testing"

	"github.com/huandu/go-assert"
)

type testType struct {
	Foo    string
	Bar    map[string]interface{}
	Player []float64
}

type testSimple struct {
	Foo int
	Bar string
}

func TestWrap(t *testing.T) {
	a := assert.New(t)
	a.Equal(Wrap(nil), nil)

	orig := &testType{
		Foo: "abcd",
		Bar: map[string]interface{}{
			"def": 123,
			"ghi": 78.9,
		},
		Player: []float64{
			12.3, 45.6, -78.9,
		},
	}
	wrapped := Wrap(orig).(*testType)
	a.Use(&orig, &wrapped)

	a.Equal(orig, wrapped)
	a.Equal(Wrap(wrapped), wrapped)

	wrapped.Foo = "xyz"
	wrapped.Bar["ghi"] = 98.7
	wrapped.Player[1] = 65.4

	a.Equal(orig.Foo, "abcd")
	a.Equal(orig.Bar["ghi"], 78.9)
	a.Equal(orig.Player[1], 45.6)

	actual := Unwrap(wrapped).(*testType)
	a.Assert(orig == actual)
}

func TestWrapScalaPtr(t *testing.T) {
	a := assert.New(t)
	i := 123
	c := &i
	v := Wrap(c).(*int)
	orig := Unwrap(v).(*int)
	a.Use(&a, &i, &c, &v)

	a.Assert(*v == *c)
	a.Assert(orig == c)
}

func TestWrapNonePtr(t *testing.T) {
	a := assert.New(t)
	cases := []interface{}{
		123, nil, "abcd", []string{"ghi"}, map[string]int{"xyz": 123},
	}

	for _, c := range cases {
		v := Wrap(c)
		a.Equal(c, v)
	}
}

func TestUnwrapValueWhichIsNotWrapped(t *testing.T) {
	a := assert.New(t)
	s := &testType{
		Foo: "abcd",
		Bar: map[string]interface{}{
			"def": 123,
			"ghi": 78.9,
		},
		Player: []float64{
			12.3, 45.6, -78.9,
		},
	}
	v := Unwrap(s).(*testType)
	v.Foo = "xyz"

	a.Equal(s, v)
}

func TestUnwrapPlainValueWhichIsNotWrapped(t *testing.T) {
	a := assert.New(t)
	i := 0
	cases := []interface{}{
		123, "abc", nil, &i,
	}

	for _, c := range cases {
		v := Unwrap(c)

		a.Equal(c, v)

		old := c
		Undo(c)

		a.Equal(c, old)
	}
}

func TestUndo(t *testing.T) {
	a := assert.New(t)
	orig := &testType{
		Foo: "abcd",
		Bar: map[string]interface{}{
			"def": 123,
			"ghi": 78.9,
		},
		Player: []float64{
			12.3, 45.6, -78.9,
		},
	}
	wrapped := Wrap(orig).(*testType)
	a.Use(&orig, &wrapped)

	wrapped.Foo = "xyz"
	wrapped.Bar["ghi"] = 98.7
	wrapped.Player[1] = 65.4

	a.Equal(orig.Foo, "abcd")
	a.Equal(orig.Bar["ghi"], 78.9)
	a.Equal(orig.Player[1], 45.6)

	Undo(wrapped)
	a.Equal(orig, wrapped)
}
