// Copyright 2019 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

package clone

import (
	"bytes"
	"container/list"
	"io"
	"reflect"
	"testing"
	"unsafe"

	"github.com/huandu/go-assert"
)

var testFuncMap = map[string]func(t *testing.T, allocator *Allocator){
	"Basic Clone":                        testClone,
	"Slowly linked list":                 testSlowlyLinkedList,
	"Slowly cycle linked list":           testSlowlyCycleLinkedList,
	"Slowly fix invalid cycle pointers":  testSlowlyFixInvalidCyclePointers,
	"Slowly fix invalid linked pointers": testSlowlyFixInvalidLinkedPointers,
	"Clone array":                        testCloneArray,
	"Clone map":                          testCloneMap,
	"Clone bytes buffer":                 testCloneBytesBuffer,
	"Clone unexported fields":            testCloneUnexportedFields,
	"Clone unexported struct method":     testCloneUnexportedStructMethod,
	"Clone reflect type":                 testCloneReflectType,
	"Clone with skip fields":             testCloneSkipFields,
}

type T struct {
	Foo int
	Bar map[string]interface{}
}

func testClone(t *testing.T, allocator *Allocator) {
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
			v1 = clone(allocator, c)
		}

		v2 = cloneSlowly(allocator, c)
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

func testSlowlyLinkedList(t *testing.T, allocator *Allocator) {
	a := assert.New(t)
	l := list.New()
	l.PushBack("v1")
	l.PushBack("v2")
	cloned := cloneSlowly(allocator, l).(*list.List)

	a.Equal(l.Len(), cloned.Len())
	a.Equal(l.Front().Value, cloned.Front().Value)
	a.Equal(l.Back().Value, cloned.Back().Value)

	// There must be only two elements in cloned.
	a.Equal(cloned.Back(), cloned.Front().Next())
	a.Equal(cloned.Back().Next(), nil)
}

type cycleLinkedList struct {
	elems []*list.Element
	elem  *list.Element

	list *list.List
}

func testSlowlyCycleLinkedList(t *testing.T, allocator *Allocator) {
	a := assert.New(t)
	l := list.New()
	elem := l.PushBack("123")
	cycle := &cycleLinkedList{
		elems: []*list.Element{elem},
		elem:  elem,
		list:  l,
	}
	cloned := cloneSlowly(allocator, cycle).(*cycleLinkedList)

	a.Equal(l.Len(), cloned.list.Len())
	a.Equal(elem.Value, cloned.list.Front().Value)

	// There must be only one element in cloned.
	a.Equal(cloned.list.Front(), cloned.list.Back())
	a.Equal(cloned.list.Front().Next(), nil)
	a.Equal(cloned.list.Back().Next(), nil)
}

type cycleList struct {
	root cycleElement
	elem *cycleElement
}

type cycleElement struct {
	next *cycleElement
	list *cycleList
}

type cycleComplex struct {
	ch           chan bool
	scalar       int
	scalarArray  *[1]int
	scalarSlice  []string
	scalarStruct *reflect.Value

	_ []*cycleElement
	_ map[*cycleElement]*cycleElement
	_ interface{}

	array          [2]*cycleElement
	slice          []*cycleElement
	iface1, iface2 interface{}
	ptr1, ptr2     *cycleElement

	scalarMap  map[string]int
	plainMap   map[int]*cycleElement
	simpleMap  map[*cycleList]*cycleElement
	complexMap map[*cycleElement]*cycleElement

	pair      cycleElementPair
	pairValue interface{}

	refSlice      *[]*cycleElement
	refComplexMap *map[*cycleElement]*cycleElement
}

type cycleElementPair struct {
	elem1, elem2 *cycleElement
}

func makeCycleElement() *cycleElement {
	list := &cycleList{}
	elem := &cycleElement{
		next: &list.root,
		list: list,
	}
	list.root.next = elem
	list.root.list = list
	list.elem = elem
	return &list.root
}

func (elem *cycleElement) validateCycle(t *testing.T) {
	a := assert.New(t)

	// elem is the &list.root.
	a.Assert(elem == &elem.list.root)
	a.Assert(elem.next == elem.list.elem)
	a.Assert(elem.next.next == elem)
}

func testSlowlyFixInvalidCyclePointers(t *testing.T, allocator *Allocator) {
	var scalarArray [1]int
	scalarStruct := reflect.ValueOf(1)
	value := &cycleComplex{
		ch:           make(chan bool),
		scalar:       123,
		scalarArray:  &scalarArray,
		scalarSlice:  []string{"hello"},
		scalarStruct: &scalarStruct,

		array:  [2]*cycleElement{makeCycleElement(), makeCycleElement()},
		slice:  []*cycleElement{makeCycleElement(), makeCycleElement()},
		iface1: makeCycleElement(),
		iface2: makeCycleElement(),
		ptr1:   makeCycleElement(),
		ptr2:   makeCycleElement(),

		scalarMap: map[string]int{
			"foo": 123,
		},
		plainMap: map[int]*cycleElement{
			123: makeCycleElement(),
		},
		simpleMap: map[*cycleList]*cycleElement{
			makeCycleElement().list: makeCycleElement(),
		},
		complexMap: map[*cycleElement]*cycleElement{
			makeCycleElement(): makeCycleElement(),
		},
	}
	value.refSlice = &value.slice
	value.refComplexMap = &value.complexMap
	cloned := cloneSlowly(allocator, value).(*cycleComplex)

	cloned.array[0].validateCycle(t)
	cloned.array[1].validateCycle(t)
	cloned.slice[0].validateCycle(t)
	cloned.slice[1].validateCycle(t)

	cloned.iface1.(*cycleElement).validateCycle(t)
	cloned.iface2.(*cycleElement).validateCycle(t)
	cloned.ptr1.validateCycle(t)
	cloned.ptr2.validateCycle(t)
	cloned.plainMap[123].validateCycle(t)

	for k, v := range cloned.simpleMap {
		k.root.validateCycle(t)
		k.elem.next.validateCycle(t)
		v.validateCycle(t)
	}

	for k, v := range cloned.complexMap {
		k.validateCycle(t)
		v.validateCycle(t)
	}

	a := assert.New(t)
	a.Assert(cloned.refSlice == &cloned.slice)
	a.Assert(cloned.refComplexMap == &cloned.complexMap)
}

func makeLinkedElements() (elem1, elem2 *cycleElement) {
	list := &cycleList{}
	elem1 = &list.root
	elem2 = &cycleElement{
		next: &list.root,
		list: list,
	}
	list.root.next = &cycleElement{}
	list.elem = elem2

	return
}

func (elem *cycleElement) validateLinked(t *testing.T) {
	a := assert.New(t)

	// elem is the elem2.
	a.Assert(elem == elem.list.elem)
	a.Assert(elem.next == &elem.list.root)
	a.Assert(elem.next.next.next == nil)
}

func testSlowlyFixInvalidLinkedPointers(t *testing.T, allocator *Allocator) {
	value := &cycleComplex{
		array: func() (elems [2]*cycleElement) {
			elems[0], elems[1] = makeLinkedElements()
			return
		}(),
		slice: func() []*cycleElement {
			elem1, elem2 := makeLinkedElements()
			return []*cycleElement{elem1, elem2}
		}(),

		scalarMap: map[string]int{
			"foo": 123,
		},
		plainMap: func() map[int]*cycleElement {
			elem1, elem2 := makeLinkedElements()
			return map[int]*cycleElement{
				1: elem1,
				2: elem2,
			}
		}(),
		simpleMap: func() map[*cycleList]*cycleElement {
			elem1, elem2 := makeLinkedElements()
			return map[*cycleList]*cycleElement{
				elem2.list: elem1,
			}
		}(),
		complexMap: func() map[*cycleElement]*cycleElement {
			elem1, elem2 := makeLinkedElements()
			return map[*cycleElement]*cycleElement{
				elem1: elem2,
			}
		}(),
	}
	value.refSlice = &value.slice
	value.refComplexMap = &value.complexMap
	value.iface1, value.iface2 = makeLinkedElements()
	value.ptr1, value.ptr2 = makeLinkedElements()
	value.pair.elem1, value.pair.elem2 = makeLinkedElements()
	var pair cycleElementPair
	pair.elem1, pair.elem2 = makeLinkedElements()
	value.pairValue = pair
	cloned := cloneSlowly(allocator, value).(*cycleComplex)

	cloned.array[1].validateLinked(t)
	cloned.slice[1].validateLinked(t)

	cloned.iface2.(*cycleElement).validateLinked(t)
	cloned.ptr2.validateLinked(t)
	cloned.plainMap[2].validateLinked(t)

	for k := range cloned.simpleMap {
		k.elem.validateLinked(t)
	}

	for _, v := range cloned.complexMap {
		v.validateLinked(t)
	}

	value.pair.elem2.validateLinked(t)
	value.pairValue.(cycleElementPair).elem2.validateLinked(t)

	a := assert.New(t)
	a.Assert(cloned.refSlice == &cloned.slice)
	a.Assert(cloned.refComplexMap == &cloned.complexMap)
}

func testCloneArray(t *testing.T, allocator *Allocator) {
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
	cloned := clone(allocator, arr).([2]*T)
	a.Use(&arr, &cloned)

	a.Equal(arr, cloned)

	// arr is not changed if cloned is mutated.
	cloned[0].Foo = 987
	cloned[1].Bar["ghi"] = 321
	a.Equal(arr[0].Foo, 123)
	a.Equal(arr[1].Bar["ghi"], 789)
}

func testCloneMap(t *testing.T, allocator *Allocator) {
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
	cloned := clone(allocator, m).(map[string]*T)
	a.Use(&m, &cloned)

	a.Equal(m, cloned)

	// m is not changed if cloned is mutated.
	cloned["abc"].Foo = 321
	cloned["def"].Bar["def"] = 987
	a.Equal(m["abc"].Foo, 123)
	a.Equal(m["def"].Bar["def"], 789)
}

func testCloneBytesBuffer(t *testing.T, allocator *Allocator) {
	a := assert.New(t)
	buf := &bytes.Buffer{}
	buf.WriteString("Hello, world!")
	dummy := make([]byte, len("Hello, "))
	buf.Read(dummy)
	cloned := clone(allocator, buf).(*bytes.Buffer)
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
	ifaceScalar   io.Writer
	_             interface{}
	m             map[string]interface{}
	ptr           *Unexported
	_             *Unexported
	slice         []*Unexported
	st            Simple
	unsafePointer unsafe.Pointer
	t             reflect.Type

	Simple
}

type scalarWriter int8

func (scalarWriter) Write(p []byte) (n int, err error) { return }

func testCloneUnexportedFields(t *testing.T, allocator *Allocator) {
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
			method:      bytes.NewBufferString("method").Write,
			iface:       bytes.NewBufferString("interface"),
			ifaceScalar: scalarWriter(123),
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
	cloned := cloneSlowly(allocator, unexported).(*Unexported)
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

func testCloneUnexportedStructMethod(t *testing.T, allocator *Allocator) {
	a := assert.New(t)

	// Another complex case: clone a struct and a map of struct instead of ptr to a struct.
	st := insider{
		m: map[string]interface{}{
			"insider": insider{
				method: bytes.NewBufferString("method").Write,
			},
		},
	}
	cloned := clone(allocator, st).(insider)
	a.Use(&st, &cloned)

	// For a struct copy, there is a tricky way to copy method. Test it.
	a.Assert(cloned.m["insider"].(insider).method != nil)
	n, err := cloned.m["insider"].(insider).method([]byte("1234"))
	a.NilError(err)
	a.Equal(n, 4)
}

func testCloneReflectType(t *testing.T, allocator *Allocator) {
	a := assert.New(t)

	// reflect.rtype should not be deeply cloned.
	foo := reflect.TypeOf("foo")
	cloned := clone(allocator, foo).(reflect.Type)
	a.Use(&foo, &cloned)

	from := reflect.ValueOf(foo)
	to := reflect.ValueOf(cloned)

	a.Assert(from.Pointer() == to.Pointer())
}

const testBytes = 332

type skipFields struct {
	Int               int
	IntSkip           int `clone:"-"`
	privateInt64      int64
	privateUint64Skip uint64 `clone:"skip"`
	str               string
	StrSkip           string `clone:"-"`
	privateStrSkip    string `clone:"-"`
	float             float32
	FloatSkip         float32 `clone:"-"`
	privateFloatSkip  float64 `clone:"-"`
	t                 *T
	TSkip             *T `clone:"-"`
	privateTSkip      *T `clone:"-"`
	privateTShadow    *T `clone:"shadowcopy"`
	privateTSlice     []*T
	privateTSliceSkip []*T `clone:"-"`
	bytes             [testBytes]byte
	bytesSkip         [testBytes]byte `clone:"-"`
}

func testCloneSkipFields(t *testing.T, allocator *Allocator) {
	a := assert.New(t)

	from := &skipFields{
		Int:               123,
		IntSkip:           456,
		privateInt64:      789,
		privateUint64Skip: 987,
		str:               "abc",
		StrSkip:           "def",
		privateStrSkip:    "ghi",
		float:             3.2,
		FloatSkip:         6.4,
		privateFloatSkip:  9.6,
		t: &T{
			Foo: 123,
			Bar: map[string]interface{}{
				"abc": 123,
			},
		},
		TSkip: &T{
			Foo: 456,
			Bar: map[string]interface{}{
				"def": 456,
				"ghi": 789,
			},
		},
		privateTSkip: &T{
			Foo: 789,
			Bar: map[string]interface{}{
				"jkl": 987,
				"mno": 654,
			},
		},
		privateTShadow: &T{
			Foo: 321,
			Bar: map[string]interface{}{
				"pqr": 321,
				"stu": 654,
			},
		},
		privateTSlice: []*T{
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
		},
		privateTSliceSkip: []*T{
			{
				Foo: 789,
				Bar: map[string]interface{}{
					"jkl": 987,
					"mno": 654,
				},
			},
			{
				Foo: 321,
				Bar: map[string]interface{}{
					"pqr": 321,
					"stu": 654,
				},
			},
		},
	}

	for i := 0; i < testBytes; i++ {
		from.bytes[i] = byte(3 + (i % 128))
		from.bytesSkip[i] = byte(3 + (i % 128))
	}

	to := clone(allocator, from).(*skipFields)

	a.Equal(from.Int, to.Int)
	a.Equal(to.IntSkip, int(0))
	a.Equal(from.privateInt64, to.privateInt64)
	a.Equal(to.privateUint64Skip, uint64(0))
	a.Equal(from.str, to.str)
	a.Equal(to.StrSkip, "")
	a.Equal(to.privateStrSkip, "")
	a.Equal(from.float, to.float)
	a.Equal(to.FloatSkip, float32(0))
	a.Equal(to.privateFloatSkip, float64(0))
	a.Equal(from.t, to.t)
	a.Equal(to.TSkip, (*T)(nil))
	a.Equal(to.privateTSkip, (*T)(nil))
	a.Assert(from.privateTShadow == to.privateTShadow)
	a.Equal(from.privateTSlice, to.privateTSlice)
	a.Equal(to.privateTSliceSkip, ([]*T)(nil))
	a.Equal(from.bytes, to.bytes)
	a.Equal(to.bytesSkip, [testBytes]byte{})
}
