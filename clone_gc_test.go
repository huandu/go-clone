// Copyright 2026 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

package clone

import (
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"unsafe"
)

// The tests in this file cover the paths that write cloned values with typed
// stores (pointer-bearing data) or raw byte copies (pointer-free data).
// Each test runs a GC before it checks the clone, so a clone that lost a value
// to the collector shows up as a wrong or corrupted field.
//
// These tests can't prove that every store runs the write barrier: a missed
// barrier only loses data in a narrow timing window that tests can't hit on
// purpose. TestHasPointers covers the choice between the two paths instead.

var cloneFuncsForTest = []struct {
	name string
	fn   func(interface{}) interface{}
}{
	{"Clone", Clone},
	{"Slowly", Slowly},
}

func stringData(s string) unsafe.Pointer {
	return (*stringHeader)(unsafe.Pointer(&s)).Data
}

type skipAllKinds struct {
	keep     *int
	n        int
	small    int        `clone:"-"`
	ptr      *int       `clone:"-"`
	str      string     `clone:"-"`
	bigBytes [2048]byte `clone:"-"` // Over 1 KiB, no pointers: cleared as raw bytes.
	bigPtrs  [160]*int  `clone:"-"` // Over 1 KiB, with pointers: cleared by a typed store.
}

// onlySkips has skipped fields but no field that needs a deep clone.
type onlySkips struct {
	n   int
	big [2048]byte `clone:"-"`
	s   string     `clone:"-"`
}

type skipHolder struct {
	val   skipAllKinds
	ptr   *skipAllKinds
	arr   [2]skipAllKinds
	slice []skipAllKinds
	m     map[int]skipAllKinds
	iface interface{}
	only  onlySkips
	onlys []onlySkips
}

func newSkipAllKinds(v int) skipAllKinds {
	p := &v
	s := skipAllKinds{keep: p, n: v, small: v, ptr: p, str: strconv.Itoa(v)}

	for i := range s.bigBytes {
		s.bigBytes[i] = byte(v) | 1
	}

	for i := range s.bigPtrs {
		s.bigPtrs[i] = p
	}

	return s
}

func newOnlySkips(v int) onlySkips {
	s := onlySkips{n: v, s: strconv.Itoa(v)}

	for i := range s.big {
		s.big[i] = 1
	}

	return s
}

func checkSkipAllKinds(t *testing.T, where string, got, src skipAllKinds) {
	t.Helper()

	if got.keep == nil || got.keep == src.keep || *got.keep != *src.keep || got.n != src.n {
		t.Errorf("%s: kept fields are not deep cloned", where)
	}

	if got.small != 0 || got.ptr != nil || got.str != "" ||
		got.bigBytes != ([2048]byte{}) || got.bigPtrs != ([160]*int{}) {
		t.Errorf("%s: skipped fields are not zero", where)
	}

	if src.small == 0 || src.ptr == nil || src.str == "" || src.bigBytes[0] == 0 || src.bigPtrs[0] == nil {
		t.Errorf("%s: skipped fields of the source were changed", where)
	}
}

func checkOnlySkips(t *testing.T, where string, got, src onlySkips) {
	t.Helper()

	if got.n != src.n || got.big != ([2048]byte{}) || got.s != "" {
		t.Errorf("%s: struct with only skipped fields is cloned wrong: n=%d s=%q", where, got.n, got.s)
	}
}

// TestCloneSkipFieldsInContainers checks that skipped fields are zero wherever
// the struct appears: by value, behind a pointer, in arrays, slices, maps and
// interfaces. Map values and interface elements are not addressable in
// reflect, so they take a different path than fields behind a pointer.
func TestCloneSkipFieldsInContainers(t *testing.T) {
	ptr := newSkipAllKinds(2)
	src := skipHolder{
		val:   newSkipAllKinds(1),
		ptr:   &ptr,
		arr:   [2]skipAllKinds{newSkipAllKinds(3), newSkipAllKinds(4)},
		slice: []skipAllKinds{newSkipAllKinds(5)},
		m:     map[int]skipAllKinds{6: newSkipAllKinds(6)},
		iface: newSkipAllKinds(7),
		only:  newOnlySkips(8),
		onlys: []onlySkips{newOnlySkips(9)},
	}

	for _, cf := range cloneFuncsForTest {
		for _, byPtr := range []bool{false, true} {
			where := fmt.Sprintf("%s(byPtr=%v)", cf.name, byPtr)
			var got skipHolder

			if byPtr {
				got = *cf.fn(&src).(*skipHolder)
			} else {
				got = cf.fn(src).(skipHolder)
			}

			runtime.GC()

			checkSkipAllKinds(t, where+".val", got.val, src.val)
			checkSkipAllKinds(t, where+".ptr", *got.ptr, *src.ptr)
			checkSkipAllKinds(t, where+".arr[0]", got.arr[0], src.arr[0])
			checkSkipAllKinds(t, where+".arr[1]", got.arr[1], src.arr[1])
			checkSkipAllKinds(t, where+".slice[0]", got.slice[0], src.slice[0])
			checkSkipAllKinds(t, where+".m[6]", got.m[6], src.m[6])
			checkSkipAllKinds(t, where+".iface", got.iface.(skipAllKinds), src.iface.(skipAllKinds))
			checkOnlySkips(t, where+".only", got.only, src.only)
			checkOnlySkips(t, where+".onlys[0]", got.onlys[0], src.onlys[0])
		}
	}
}

// privateScalarSlices holds slices whose element kinds are scalar to the
// default allocator. ints has no pointers and is copied as raw bytes; the
// other slices hold pointers and must be copied with typed stores.
type privateScalarSlices struct {
	ints  []int
	strs  []string
	funcs []func() int
	ptrs  []unsafe.Pointer
	arr   [3]string
}

type scalarSliceHolder struct {
	val privateScalarSlices
	ptr *privateScalarSlices
	m   map[int]privateScalarSlices
}

const scalarSliceLen = 8

func newPrivateScalarSlices(base int) privateScalarSlices {
	var s privateScalarSlices

	for i := 0; i < scalarSliceLen; i++ {
		v := base + i
		s.ints = append(s.ints, v)
		s.strs = append(s.strs, strconv.Itoa(v))
		s.funcs = append(s.funcs, func() int { return v })
		s.ptrs = append(s.ptrs, unsafe.Pointer(&v))
	}

	for i := range s.arr {
		s.arr[i] = strconv.Itoa(base*10 + i)
	}

	return s
}

func checkPrivateScalarSlices(t *testing.T, where string, got, src privateScalarSlices, base int) {
	t.Helper()

	checkScalarSliceValues(t, where, got, base)

	if &got.ints[0] == &src.ints[0] || &got.strs[0] == &src.strs[0] ||
		&got.funcs[0] == &src.funcs[0] || &got.ptrs[0] == &src.ptrs[0] {
		t.Errorf("%s: a cloned slice shares its backing array with the source", where)
	}
}

func checkScalarSliceValues(t *testing.T, where string, got privateScalarSlices, base int) {
	t.Helper()
	if len(got.ints) != scalarSliceLen || len(got.strs) != scalarSliceLen ||
		len(got.funcs) != scalarSliceLen || len(got.ptrs) != scalarSliceLen {
		t.Fatalf("%s: wrong slice lengths", where)
	}

	for i := 0; i < scalarSliceLen; i++ {
		v := base + i

		if got.ints[i] != v || got.strs[i] != strconv.Itoa(v) || got.funcs[i]() != v || *(*int)(got.ptrs[i]) != v {
			t.Errorf("%s: element %d is wrong", where, i)
		}
	}

	for i := range got.arr {
		if got.arr[i] != strconv.Itoa(base*10+i) {
			t.Errorf("%s: arr[%d] = %q", where, i, got.arr[i])
		}
	}
}

// TestClonePrivateScalarSlicesDuringGC is a smoke test: it clones while
// another goroutine runs GC nonstop, drops each source, and checks that the
// retained clones still hold their strings, closures and unsafe.Pointer
// elements.
//
// It does not catch a missing write barrier. Objects created during a mark
// phase start out marked, so sources created in the loop can't be freed early.
// A mutant that copies these slices as raw bytes still passes, even with
// GODEBUG=gccheckmark=1.
func TestClonePrivateScalarSlicesDuringGC(t *testing.T) {
	for _, cf := range cloneFuncsForTest {
		t.Run(cf.name, func(t *testing.T) {
			started, stop, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
			go func() {
				defer close(done)
				close(started)
				for {
					runtime.GC()
					select {
					case <-stop:
						return
					default:
					}
				}
			}()

			// Stop the GC goroutine on every exit path, including a failed
			// type assertion or an early t.Fatal.
			stopped := false
			stopGC := func() {
				if !stopped {
					stopped = true
					close(stop)
					<-done
				}
			}
			defer stopGC()

			<-started
			const base = 10000
			retained := make([]privateScalarSlices, 1024)
			for i := range retained {
				retained[i] = cf.fn(newPrivateScalarSlices(base + i*scalarSliceLen)).(privateScalarSlices)
			}
			stopGC()
			runtime.GC()
			for i, got := range retained {
				checkScalarSliceValues(t, fmt.Sprintf("clone %d", i), got, base+i*scalarSliceLen)
			}
		})
	}
}

// TestClonePrivateScalarSlices covers unexported scalar slices, reached through
// a non-addressable value, a pointer and a map value.
func TestClonePrivateScalarSlices(t *testing.T) {
	ptr := newPrivateScalarSlices(100)
	src := scalarSliceHolder{
		val: newPrivateScalarSlices(0),
		ptr: &ptr,
		m:   map[int]privateScalarSlices{1: newPrivateScalarSlices(200)},
	}

	for _, cf := range cloneFuncsForTest {
		for _, byPtr := range []bool{false, true} {
			where := fmt.Sprintf("%s(byPtr=%v)", cf.name, byPtr)
			var got scalarSliceHolder

			if byPtr {
				got = *cf.fn(&src).(*scalarSliceHolder)
			} else {
				got = cf.fn(src).(scalarSliceHolder)
			}

			runtime.GC()

			checkPrivateScalarSlices(t, where+".val", got.val, src.val, 0)
			checkPrivateScalarSlices(t, where+".ptr", *got.ptr, *src.ptr, 100)
			checkPrivateScalarSlices(t, where+".m[1]", got.m[1], src.m[1], 200)

			got.val.ints[0] = -1
			got.val.strs[0] = "changed"

			if src.val.ints[0] != 0 || src.val.strs[0] != "0" {
				t.Errorf("%s: changing the clone changed the source", where)
			}
		}
	}
}

// TestCloneNonScalarStrings covers cloneString, which copies string bytes
// into a new buffer and stores its header with a typed store. The default
// allocator treats strings as scalars, so a custom allocator is needed.
func TestCloneNonScalarStrings(t *testing.T) {
	allocator := NewAllocator(nil, &AllocatorMethods{
		IsScalar: func(k reflect.Kind) bool {
			return k != reflect.String && IsScalar(k)
		},
	})

	type texts struct {
		s    string
		list []string
		m    map[string]string
	}

	src := texts{
		s:    strings.Repeat("s", 64),
		list: []string{strings.Repeat("a", 32), strconv.Itoa(12345)},
		m:    map[string]string{strings.Repeat("k", 16): strings.Repeat("v", 16)},
	}

	for _, slowly := range []bool{false, true} {
		where := fmt.Sprintf("slowly=%v", slowly)
		v := reflect.ValueOf(src)
		var got texts

		if slowly {
			got = allocator.CloneSlowly(v).Interface().(texts)
		} else {
			got = allocator.Clone(v).Interface().(texts)
		}

		runtime.GC()

		if got.s != src.s || len(got.list) != len(src.list) || got.list[0] != src.list[0] ||
			got.list[1] != src.list[1] || got.m[strings.Repeat("k", 16)] != strings.Repeat("v", 16) {
			t.Fatalf("%s: cloned strings differ from the source: %+v", where, got)
		}

		if stringData(got.s) == stringData(src.s) || stringData(got.list[0]) == stringData(src.list[0]) {
			t.Errorf("%s: cloned string shares its bytes with the source", where)
		}
	}
}
