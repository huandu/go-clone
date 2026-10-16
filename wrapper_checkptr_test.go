// Copyright 2019 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

package clone

import (
	"runtime"
	"testing"

	"github.com/huandu/go-assert"
)

// TestUnwrapAndUndoOnNeverWrappedPointers is a regression test for a checkptr
// violation: detection used to read ptr+sizeof(T)+8, which lies outside the
// allocation of any *T that Wrap never produced. Under `go test -race` (which
// enables checkptr) that aborted the process with
//
//	fatal error: checkptr: pointer arithmetic result points to invalid allocation
//
// so these calls are exercised across a spread of pointer kinds and storage
// classes, including ones that may live on the stack.
func TestUnwrapAndUndoOnNeverWrappedPointers(t *testing.T) {
	a := assert.New(t)

	i := 123
	s := "abcd"
	f := 1.5
	b := true
	st := testType{Foo: "abcd"}
	sl := []int{1, 2, 3}
	m := map[string]int{"xyz": 123}

	cases := []interface{}{
		&i,
		&s,
		&f,
		&b,
		&st,
		&sl,
		&m,
		new(int),
		new(string),
		new(testType),
		new([]int),
		new(map[string]int),
	}

	for _, c := range cases {
		// Unwrap must be the identity and must not panic.
		a.Equal(c, Unwrap(c))

		// Undo must leave the value untouched and must not panic.
		Undo(c)
		a.Equal(c, Unwrap(c))
	}

	// Values that are not pointers at all are returned untouched.
	for _, c := range []interface{}{nil, 123, "abc", sl, m} {
		a.Equal(c, Unwrap(c))
		Undo(c)
	}
}

// TestWrappedPointerDetectionOnOddTypes exercises pointer types whose element
// size is small enough that the old probe would run furthest past the end of the
// allocation, plus pointer-to-pointer and pointer-to-array cases.
func TestWrappedPointerDetectionOnOddTypes(t *testing.T) {
	a := assert.New(t)

	// Small element types: the old probe read 8 bytes past a 1/4/8 byte object.
	b := byte(7)
	a.Equal(&b, Unwrap(&b))
	Undo(&b)
	a.Assert(b == 7)

	i32 := int32(9)
	a.Equal(&i32, Unwrap(&i32))
	Undo(&i32)
	a.Assert(i32 == 9)

	// Pointer to pointer, both wrapped and plain.
	inner := 42
	p := &inner
	pp := &p
	a.Equal(pp, Unwrap(pp))
	Undo(pp)

	wrapped := Wrap(&inner).(*int)
	a.Assert(Unwrap(wrapped) == &inner)
	a.Assert(wrapped != &inner)

	// Pointer to array.
	arr := [3]int{1, 2, 3}
	a.Equal(&arr, Unwrap(&arr))
	Undo(&arr)

	wrappedArr := Wrap(&arr).(*[3]int)
	wrappedArr[0] = 99
	a.Assert(arr[0] == 1)
	Undo(wrappedArr)
	a.Equal(&arr, Unwrap(wrappedArr))
}

// TestWrapDoesNotLeakRegistryEntries verifies that the address registry used to
// detect wrapped values does not prevent the wrapper (or the value it wraps)
// from being collected, and that entries are dropped once a wrapper is
// unreachable.
//
// Finalizer scheduling is asynchronous and `go test -race` amplifies the delay a
// lot: measured on Go 1.26.2 with this test, a plain run converges to 0 entries
// while a race-instrumented run settles at roughly half the batch. The property
// asserted here is therefore boundedness rather than prompt cleanup, which is
// what actually matters and is stable across both modes: two batches of the same
// size must not retain twice as many entries. A registry kept alive by its own
// map, or one whose finalizer never fires, would instead retain one entry per
// wrapper ever created and the second batch would double the count.
func TestWrapDoesNotLeakRegistryEntries(t *testing.T) {
	a := assert.New(t)

	registrySize := func() int {
		n := 0
		wrapped.Range(func(key, value interface{}) bool {
			n++
			return true
		})
		return n
	}

	const batch = 1000

	wrapBatch := func() {
		for i := 0; i < batch; i++ {
			v := Wrap(&testType{Foo: "temp"}).(*testType)
			runtime.KeepAlive(v)
		}
		// Several GC cycles are needed because each one may only run the
		// finalizers queued by the previous collection.
		for i := 0; i < 6; i++ {
			runtime.GC()
			runtime.Gosched()
		}
	}

	// Baseline: wrapping something must register exactly one address.
	kept := Wrap(&testType{Foo: "keep"}).(*testType)
	a.Assert(registrySize() >= 1)

	wrapBatch()
	first := registrySize()

	wrapBatch()
	second := registrySize()

	t.Logf("registry entries: after batch 1 = %d, after batch 2 = %d (batch size %d)", first, second, batch)

	// If entries were never removed, the second batch would add another `batch`
	// entries and this would be around 2*batch.
	if second-first >= batch {
		t.Errorf("registry grew by %d entries for a batch of %d; wrappers are never unregistered", second-first, batch)
	}
	if second >= 2*batch {
		t.Errorf("registry holds %d entries after %d wraps; entries are not being removed", second, 2*batch)
	}

	// The wrapped value we still hold must remain detected.
	a.Assert(Unwrap(kept) != kept)
	a.Assert(Unwrap(kept).(*testType).Foo == "keep")
}

// TestWrapOfWrappedValue pins the documented behavior that wrapping an already
// wrapped value yields a value equal to it, and that the wrapper keeps working
// after the re-wrap. This mirrors the existing assertion in TestWrap, which
// compares with assert.Equal rather than pointer identity.
func TestWrapOfWrappedValue(t *testing.T) {
	a := assert.New(t)

	orig := &testType{Foo: "abcd"}
	wrapped := Wrap(orig).(*testType)

	// Re-wrapping is not identity: the wrapper is deep cloned like any value.
	again := Wrap(wrapped).(*testType)
	a.Equal(wrapped, again)

	// The original wrapper is untouched and still recognized.
	a.Assert(Unwrap(wrapped) == orig)

	// One Unwrap peels exactly one layer.
	a.Assert(Unwrap(again) == wrapped)
}
