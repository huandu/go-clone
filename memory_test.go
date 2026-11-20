// Copyright 2026 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

package clone

import (
	"bytes"
	"reflect"
	"testing"
	"unsafe"
)

func TestHasPointers(t *testing.T) {
	cases := []struct {
		typ  reflect.Type
		want bool
	}{
		{reflect.TypeOf(int(0)), false},
		{reflect.TypeOf(uintptr(0)), false},
		{reflect.TypeOf(complex128(0)), false},
		{reflect.TypeOf([4]int{}), false},
		{reflect.TypeOf([0]*int{}), false},
		{reflect.TypeOf(struct {
			a int
			b [2]float64
		}{}), false},
		{reflect.TypeOf(struct{}{}), false},

		{reflect.TypeOf(""), true},
		{reflect.TypeOf([]int(nil)), true},
		{reflect.TypeOf((*int)(nil)), true},
		{reflect.TypeOf(map[int]int(nil)), true},
		{reflect.TypeOf((chan int)(nil)), true},
		{reflect.TypeOf((func())(nil)), true},
		{reflect.TypeOf(unsafe.Pointer(nil)), true},
		{reflect.TypeOf((*interface{})(nil)).Elem(), true},
		{reflect.TypeOf([2]string{}), true},
		{reflect.TypeOf(struct {
			a int
			p *int
		}{}), true},
		{reflect.TypeOf(struct{ s struct{ b []byte } }{}), true},
	}

	for _, c := range cases {
		if got := hasPointers(c.typ); got != c.want {
			t.Errorf("hasPointers(%v) = %v, want %v", c.typ, got, c.want)
		}
	}
}

func TestClearAndCopyBytes(t *testing.T) {
	const size, from, n = 3000, 10, 2000

	fill := func() []byte {
		buf := make([]byte, size)

		for i := range buf {
			buf[i] = byte(i%251) + 1
		}

		return buf
	}

	buf := fill()
	clearBytes(unsafe.Pointer(&buf[from]), n)

	for i, b := range buf {
		inRange := i >= from && i < from+n

		if inRange && b != 0 {
			t.Fatalf("clearBytes: byte %d = %d, want 0", i, b)
		}

		if !inRange && b != byte(i%251)+1 {
			t.Fatalf("clearBytes: byte %d outside the range was changed", i)
		}
	}

	src := fill()
	dst := make([]byte, size)
	copyBytes(unsafe.Pointer(&dst[from]), unsafe.Pointer(&src[from]), n)

	for i, b := range dst {
		want := byte(0)

		if i >= from && i < from+n {
			want = src[i]
		}

		if b != want {
			t.Fatalf("copyBytes: byte %d = %d, want %d", i, b, want)
		}
	}

	// A range that ends exactly at the end of an allocation must not move a
	// pointer past it; checkptr (on with -race) fails the test if it does.
	tailSrc := fill()
	tailDst := make([]byte, size)
	copyBytes(unsafe.Pointer(&tailDst[from]), unsafe.Pointer(&tailSrc[from]), size-from)
	clearBytes(unsafe.Pointer(&tailSrc[from]), size-from)

	if tailDst[size-1] != byte((size-1)%251)+1 || tailSrc[size-1] != 0 {
		t.Fatal("copyBytes or clearBytes did not reach the end of the buffer")
	}

	// Zero length must not touch memory.
	clearBytes(unsafe.Pointer(&src[0]), 0)
	copyBytes(unsafe.Pointer(&dst[0]), unsafe.Pointer(&src[0]), 0)

	if src[0] != 1 || dst[0] != 0 {
		t.Fatal("zero-length clearBytes or copyBytes changed memory")
	}
}

func TestCopyAndClearChunkBoundaries(t *testing.T) {
	const chunk = 8
	for _, n := range []int{chunk - 1, chunk, chunk + 1, 3 * chunk, 3*chunk + 2} {
		// Distinct offsets and guard bytes expose incorrect pointer advancement.
		src := make([]byte, n+3)
		for i := range src {
			src[i] = byte(i + 1)
		}
		dst := bytes.Repeat([]byte{0xff}, n+4)
		want := append([]byte(nil), dst...)
		copy(want[2:2+n], src[1:1+n])
		copyBytesInChunks(unsafe.Pointer(&dst[2]), unsafe.Pointer(&src[1]), uintptr(n), chunk)
		if !bytes.Equal(dst, want) {
			t.Fatalf("copy of %d bytes: got %x, want %x", n, dst, want)
		}

		for i := 2; i < 2+n; i++ {
			want[i] = 0
		}
		clearBytesInChunks(unsafe.Pointer(&dst[2]), uintptr(n), chunk)
		if !bytes.Equal(dst, want) {
			t.Fatalf("clear of %d bytes: got %x, want %x", n, dst, want)
		}
	}
}

// chunkTestSink forces the buffers in TestCopyAndClearChunksToAllocationEnd
// onto the heap, where checkptr knows the bounds of each allocation.
var chunkTestSink [][]byte

// TestCopyAndClearChunksToAllocationEnd copies and clears ranges that span
// several chunks and end exactly at the end of their heap allocation. If a
// helper advanced its pointer past the last byte, checkptr (on with -race)
// would fail the test. The sizes are malloc size classes, so the end of the
// slice is the end of the allocation.
func TestCopyAndClearChunksToAllocationEnd(t *testing.T) {
	const chunk = 8

	for _, n := range []int{48, 64, 4096} {
		src := make([]byte, n)
		dst := make([]byte, n)
		chunkTestSink = append(chunkTestSink[:0], src, dst)

		for i := range src {
			src[i] = byte(i%251) + 1
		}

		copyBytesInChunks(unsafe.Pointer(&dst[0]), unsafe.Pointer(&src[0]), uintptr(n), chunk)

		if !bytes.Equal(dst, src) {
			t.Fatalf("copy of %d bytes to the allocation end: got %x, want %x", n, dst, src)
		}

		clearBytesInChunks(unsafe.Pointer(&dst[0]), uintptr(n), chunk)

		if !bytes.Equal(dst, make([]byte, n)) {
			t.Fatalf("clear of %d bytes to the allocation end: got %x", n, dst)
		}
	}

	chunkTestSink = nil
}
