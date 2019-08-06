// Copyright 2019 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

package clone

import (
	"fmt"
	"reflect"
	"testing"
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
	if Wrap(nil) != nil {
		t.Fatalf("nil should not be wrapped.")
	}

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

	if !reflect.DeepEqual(orig, wrapped) {
		t.Fatalf("orig and wrapped must be the same value. [orig:%#v] [wrapped:%#v]", orig, wrapped)
	}

	if again := Wrap(wrapped); !reflect.DeepEqual(again, wrapped) {
		t.Fatalf("again and wrapped must be the same value. [again:%#v] [wrapped:%#v]", again, wrapped)
	}

	wrapped.Foo = "xyz"
	wrapped.Bar["ghi"] = 98.7
	wrapped.Player[1] = 65.4

	if orig.Foo != "abcd" || orig.Bar["ghi"] != 78.9 || orig.Player[1] != 45.6 {
		t.Fatalf("original value should be untouched. [orig:%#v]", orig)
	}

	actual := Unwrap(wrapped).(*testType)

	if orig != actual {
		t.Fatalf("fail to get original value. [expected:%#v] [actual:%#v]", orig, actual)
	}
}

func TestWrapScalaPtr(t *testing.T) {
	i := 123
	c := &i

	v := Wrap(c).(*int)

	if *v != *c {
		t.Fatalf("c and v must be the same value. [c:%#v] [v:%#v]", c, v)
	}

	orig := Unwrap(v).(*int)

	if orig != c {
		t.Fatalf("fail to get original value. [expected:%p] [actual:%p]", c, orig)
	}
}

func TestWrapNonePtr(t *testing.T) {
	cases := []interface{}{
		123, nil, "abcd", []string{"ghi"}, map[string]int{"xyz": 123},
	}

	for _, c := range cases {
		v := Wrap(c)

		if !reflect.DeepEqual(c, v) {
			t.Fatalf("c and v must be the same value. [c:%#v] [v:%#v]", c, v)
		}
	}
}

func TestUnwrapValueWhichIsNotWrapped(t *testing.T) {
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

	if !reflect.DeepEqual(s, v) {
		t.Fatalf("origin should return old value. [expected:%v] [actual:%#v]", s, v)
	}
}

func TestUnwrapPlainValueWhichIsNotWrapped(t *testing.T) {
	i := 0
	cases := []interface{}{
		123, "abc", nil, &i,
	}

	for _, c := range cases {
		v := Unwrap(c)

		if !reflect.DeepEqual(c, v) {
			t.Fatalf("origin should return old value. [expected:%#v] [actual:%#v]", c, v)
		}

		old := c
		Undo(c)

		if !reflect.DeepEqual(c, old) {
			t.Fatalf("undo should return old value. [expected:%#v] [actual:%#v]", c, old)
		}
	}
}

func BenchmarkUnwrap(b *testing.B) {
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
	wrapped := Wrap(orig)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		Unwrap(wrapped)
	}
}

func TestUndo(t *testing.T) {
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
	wrapped.Foo = "xyz"
	wrapped.Bar["ghi"] = 98.7
	wrapped.Player[1] = 65.4

	if orig.Foo != "abcd" || orig.Bar["ghi"] != 78.9 || orig.Player[1] != 45.6 {
		t.Fatalf("original value should be untouched. [orig:%#v]", orig)
	}

	Undo(wrapped)

	if !reflect.DeepEqual(orig, wrapped) {
		t.Fatalf("fail to get original value. [expected:%#v] [actual:%#v]", orig, wrapped)
	}
}

func BenchmarkSimpleWrap(b *testing.B) {
	orig := &testSimple{
		Foo: 123,
		Bar: "abcd",
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		Wrap(orig)
	}
}

func BenchmarkComplexWrap(b *testing.B) {
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
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		Wrap(orig)
	}
}

func ExampleWrap() {
	// Suppose we have a type T defined as following.
	//     type T struct {
	//         Foo int
	//     }
	v := &T{
		Foo: 123,
	}
	w := Wrap(v).(*T) // Wrap value to protect it.

	// Use w freely. The type of w is the same as that of v.

	// It's OK to modify w. The change will not affect v.
	w.Foo = 456
	fmt.Println(w.Foo) // 456
	fmt.Println(v.Foo) // 123

	// Once we need the original value stored in w, call `Unwrap`.
	orig := Unwrap(w).(*T)
	fmt.Println(orig == v) // true
	fmt.Println(orig.Foo)  // 123

	// Or, we can simply undo any change made in w.
	// Note that `Undo` is significantly slower than `Unwrap`, thus
	// the latter is always preferred.
	Undo(w)
	fmt.Println(w.Foo) // 123

	// Output:
	// 456
	// 123
	// true
	// 123
	// 123
}
