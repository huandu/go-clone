// Copyright 2019 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

package clone

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"testing"
)

type T struct {
	Foo int
	Bar map[string]interface{}
}

func TestClone(t *testing.T) {
	arr := [4]string{"abc", "def", "ghi"}
	ch := make(chan int, 2)
	fn := func(int) {}
	var it io.Writer = &bytes.Buffer{}
	m := map[interface{}]string{
		"abc": "efg",
		123:   "ghi",
	}
	slice := []string{"xyz", "opq"}
	st := T{
		Foo: 1234,
		Bar: map[string]interface{}{
			"abc": 123,
			"def": "ghi",
		},
	}
	ptr := &st
	complex := []map[string][]*T{
		{
			"abc": {
				{Foo: 456, Bar: map[string]interface{}{"abc": "def"}},
			},
		},
		{
			"def": {
				{Foo: 987, Bar: map[string]interface{}{"abc": "def"}},
				{Foo: 321, Bar: map[string]interface{}{"ghi": "xyz"}},
			},
			"ghi": {
				{Foo: 654, Bar: map[string]interface{}{"def": "abc"}},
			},
		},
	}
	nested := func() interface{} {
		var nested []map[string][]*T
		var nestedPtr *T
		var nestedIf interface{}
		var nestedMap map[string]interface{}
		nested = []map[string][]*T{
			{
				"abc": {
					{Foo: 987, Bar: map[string]interface{}{"def": nil, "nil": nil}},
					{Foo: 321, Bar: map[string]interface{}{"ghi": nil, "def": nil, "cba": nil}},
					{Foo: 456},
					nil,
				},
			},
		}
		nestedPtr = &T{
			Foo: 654,
			Bar: map[string]interface{}{
				"xyz": nested,
				"opq": nil,
			},
		}
		nestedIf = map[string]interface{}{
			"rst": nested,
		}
		nestedMap = map[string]interface{}{}

		// Don't test it due to bug in Go.
		// https://github.com/golang/go/issues/33907
		//nestedMap["opq"] = nestedMap

		nested[0]["abc"][0].Bar["def"] = nested
		nested[0]["abc"][1].Bar["ghi"] = nestedPtr
		nested[0]["abc"][1].Bar["def"] = nestedIf
		nested[0]["abc"][1].Bar["cba"] = nested
		nested[0]["abc"][2].Bar = nestedMap
		nested[0]["abc"][3] = nestedPtr
		nestedPtr.Bar["opq"] = nestedPtr
		return nested
	}()
	var nilSlice []int
	var nilChan chan bool
	var nilPtr *float64
	cases := []interface{}{
		123, "abc", nil, true, testing.TB(nil),
		arr, ch, fn, it, m, ptr, slice, st, nested,
		complex, nilSlice, nilChan, nilPtr,
	}

	for _, c := range cases {
		var v1, v2 interface{}

		if reflect.DeepEqual(c, nested) {
			// Clone doesn't work on nested data.
			v1 = c
		} else {
			v1 = Clone(c)
		}

		v2 = Slowly(c)
		deepEqual(t, c, v1)
		deepEqual(t, c, v2)
	}
}

func deepEqual(t *testing.T, expected, actual interface{}) {
	val := reflect.ValueOf(actual)

	// It's not possible to compare chan value.
	if val.Kind() == reflect.Chan {
		cval := reflect.ValueOf(expected)

		if cval.Type() != val.Type() || cval.Cap() != val.Cap() {
			t.Fatalf("fail to clone chan. [expected:%#v] [actual:%#v]", expected, actual)
		}

		return
	}

	if val.Kind() == reflect.Func {
		// It's not possible to compare func value either.
		cval := reflect.ValueOf(expected)

		if cval.Type() != val.Type() {
			t.Fatalf("fail to clone func. [expected:%v] [actual:%v]", cval, val)
		}

		return
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("fail to clone. [expected:%T] [actual:%T]\nexpect: %#v\nactual: %#v",
			expected, actual, expected, actual)
	}
}

func TestCloneArray(t *testing.T) {
	arr := [2]*T{
		{
			Foo: 123,
			Bar: map[string]interface{}{
				"abc": 123,
			},
		},
		{
			Foo: 456,
			Bar: map[string]interface{}{
				"def": 456,
				"ghi": 789,
			},
		},
	}
	cloned := Clone(arr).([2]*T)
	cloned[0].Foo = 987
	cloned[1].Bar["ghi"] = 321

	if arr[0].Foo != 123 || arr[1].Bar["ghi"] != 789 {
		t.Fatalf("fail to do deep clone. [orig:%#v %#v] [cloned:%#v %#v]", arr[0], arr[1], cloned[0], cloned[1])
	}
}

func TestCloneMap(t *testing.T) {
	m := map[string]*T{
		"abc": {
			Foo: 123,
			Bar: map[string]interface{}{
				"abc": 321,
			},
		},
		"def": {
			Foo: 456,
			Bar: map[string]interface{}{
				"def": 789,
			},
		},
	}
	cloned := Clone(m).(map[string]*T)
	cloned["abc"].Foo = 321
	cloned["def"].Bar["def"] = 987

	if m["abc"].Foo != 123 || m["def"].Bar["def"] != 789 {
		t.Fatalf("fail to do deep clone. [orig:%#v] [cloned:%#v]", m, cloned)
	}
}

func ExampleSlowly() {
	type ListNode struct {
		Data int
		Next *ListNode
	}
	node1 := &ListNode{
		Data: 1,
	}
	node2 := &ListNode{
		Data: 2,
	}
	node3 := &ListNode{
		Data: 3,
	}
	node1.Next = node2
	node2.Next = node3
	node3.Next = node1

	// We must use `Slowly` to clone a circular linked list.
	node := Slowly(node1).(*ListNode)

	for i := 0; i < 10; i++ {
		fmt.Println(node.Data)
		node = node.Next
	}

	// Output:
	// 1
	// 2
	// 3
	// 1
	// 2
	// 3
	// 1
	// 2
	// 3
	// 1
}

func BenchmarkSimpleClone(b *testing.B) {
	orig := &testSimple{
		Foo: 123,
		Bar: "abcd",
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		Clone(orig)
	}
}

func BenchmarkComplexClone(b *testing.B) {
	m := map[string]*T{
		"abc": {
			Foo: 123,
			Bar: map[string]interface{}{
				"abc": 321,
			},
		},
		"def": {
			Foo: 456,
			Bar: map[string]interface{}{
				"def": 789,
			},
		},
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		Clone(m)
	}
}
