// Copyright 2019 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

package clone

import (
	"bytes"
	"io"
	"reflect"
	"testing"
	"unsafe"

	"github.com/huandu/go-assert"
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
	a := assert.New(t)
	a.Use(&expected, &actual)

	val := reflect.ValueOf(actual)

	// It's not possible to compare chan value.
	if val.Kind() == reflect.Chan {
		cval := reflect.ValueOf(expected)
		a.Equal(cval.Type(), val.Type())
		a.Equal(cval.Cap(), val.Cap())
		return
	}

	if val.Kind() == reflect.Func {
		// It's not possible to compare func value either.
		cval := reflect.ValueOf(expected)
		a.Assert(cval.Type() == val.Type())
		return
	}

	a.Equal(actual, expected)
}

func TestCloneArray(t *testing.T) {
	a := assert.New(t)
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
	a.Use(&arr, &cloned)

	a.Equal(arr, cloned)

	// arr is not changed if cloned is mutated.
	cloned[0].Foo = 987
	cloned[1].Bar["ghi"] = 321
	a.Equal(arr[0].Foo, 123)
	a.Equal(arr[1].Bar["ghi"], 789)
}

func TestCloneMap(t *testing.T) {
	a := assert.New(t)
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
	a.Use(&m, &cloned)

	a.Equal(m, cloned)

	// m is not changed if cloned is mutated.
	cloned["abc"].Foo = 321
	cloned["def"].Bar["def"] = 987
	a.Equal(m["abc"].Foo, 123)
	a.Equal(m["def"].Bar["def"], 789)
}

func TestCloneBytesBuffer(t *testing.T) {
	a := assert.New(t)
	buf := &bytes.Buffer{}
	buf.WriteString("Hello, world!")
	dummy := make([]byte, len("Hello, "))
	buf.Read(dummy)
	cloned := Clone(buf).(*bytes.Buffer)
	a.Use(&buf, &cloned)

	// Data must be cloned.
	a.Equal(buf.Len(), cloned.Len())
	a.Equal(buf.String(), cloned.String())

	// Data must not share the same address.
	from := buf.Bytes()
	to := cloned.Bytes()
	a.Assert(&from[0] != &to[0])

	buf.WriteString("!!!!!")
	a.NotEqual(buf.Len(), cloned.Len())
	a.NotEqual(buf.String(), cloned.String())
}

type Simple struct {
	Foo int
	Bar string
}

type Unexported struct {
	insider
}

type insider struct {
	i             int
	i8            int8
	i16           int16
	i32           int32
	i64           int64
	u             uint
	u8            uint8
	u16           uint16
	u32           uint32
	u64           uint64
	uptr          uintptr
	b             bool
	s             string
	f32           float32
	f64           float64
	c64           complex64
	c128          complex128
	arr           [4]string
	arrPtr        *[10]byte
	ch            chan bool
	fn            func(s string) string
	method        func([]byte) (int, error)
	iface         io.Writer
	nilIface      interface{}
	m             map[string]interface{}
	ptr           *Unexported
	nilPtr        *Unexported
	slice         []*Unexported
	st            Simple
	unsafePointer unsafe.Pointer
	t             reflect.Type

	Simple
}

func TestCloneUnexportedFields(t *testing.T) {
	a := assert.New(t)
	unexported := &Unexported{
		insider: insider{
			i:    -1,
			i8:   -8,
			i16:  -16,
			i32:  -32,
			i64:  -64,
			u:    1,
			u8:   8,
			u16:  16,
			u32:  32,
			u64:  64,
			uptr: uintptr(0xDEADC0DE),
			b:    true,
			s:    "hello",
			f32:  3.2,
			f64:  6.4,
			c64:  complex(6, 4),
			c128: complex(12, 8),
			arr: [4]string{
				"a", "b", "c", "d",
			},
			arrPtr: &[10]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			ch:     make(chan bool, 5),
			fn: func(s string) string {
				return s + ", world!"
			},
			method: bytes.NewBufferString("method").Write,
			iface:  bytes.NewBufferString("interface"),
			m: map[string]interface{}{
				"key": "value",
			},
			unsafePointer: unsafe.Pointer(&Unexported{}),
			st: Simple{
				Foo: 123,
				Bar: "bar1",
			},
			Simple: Simple{
				Foo: 456,
				Bar: "bar2",
			},
			t: reflect.TypeOf(&Simple{}),
		},
	}
	unexported.m["loop"] = &unexported.m

	// Make pointer cycles.
	unexported.ptr = unexported
	unexported.slice = []*Unexported{unexported}
	cloned := Slowly(unexported).(*Unexported)
	a.Use(&unexported, &cloned)

	// unsafe.Pointer is shadow copied.
	a.Assert(cloned.unsafePointer == unexported.unsafePointer)
	unexported.unsafePointer = nil
	cloned.unsafePointer = nil

	// chan cannot be compared, but its buffer can be verified.
	a.Equal(cap(cloned.ch), cap(unexported.ch))
	unexported.ch = nil
	cloned.ch = nil

	// fn cannot be compared, but it can be called.
	a.Equal(cloned.fn("Hello"), unexported.fn("Hello"))
	unexported.fn = nil
	cloned.fn = nil

	// method cannot be compared, but it can be called.
	a.Assert(cloned.method != nil)
	a.NilError(cloned.method([]byte("1234")))
	unexported.method = nil
	cloned.method = nil

	// cloned.m["loop"] must be exactly the same map of cloned.m.
	a.Assert(reflect.ValueOf(cloned.m["loop"]).Elem().Pointer() == reflect.ValueOf(cloned.m).Pointer())

	// Don't test this map in reflect.DeepEqual due to bug in Go.
	// https://github.com/golang/go/issues/33907
	unexported.m["loop"] = nil
	cloned.m["loop"] = nil

	// reflect.Type should be copied by value.
	a.Equal(reflect.ValueOf(cloned.t).Pointer(), reflect.ValueOf(unexported.t).Pointer())

	// Finally, everything else should equal.
	a.Equal(unexported, cloned)
}

func TestCloneUnexportedStructMethod(t *testing.T) {
	a := assert.New(t)

	// Another complex case: clone a struct and a map of struct instead of ptr to a struct.
	st := insider{
		m: map[string]interface{}{
			"insider": insider{
				method: bytes.NewBufferString("method").Write,
			},
		},
	}
	cloned := Clone(st).(insider)
	a.Use(&st, &cloned)

	// For a struct copy, there is a tricky way to copy method. Test it.
	a.Assert(cloned.m["insider"].(insider).method != nil)
	n, err := cloned.m["insider"].(insider).method([]byte("1234"))
	a.NilError(err)
	a.Equal(n, 4)
}

func TestCloneReflectType(t *testing.T) {
	a := assert.New(t)

	// reflect.rtype should not be deeply cloned.
	foo := reflect.TypeOf("foo")
	cloned := Clone(foo).(reflect.Type)
	a.Use(&foo, &cloned)

	from := reflect.ValueOf(foo)
	to := reflect.ValueOf(cloned)

	a.Assert(from.Pointer() == to.Pointer())
}
