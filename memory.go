// Copyright 2019 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

package clone

import (
	"reflect"
	"unsafe"
)

// maxByteSize is a large enough value to cheat Go compiler
// when converting unsafe address to []byte.
// It's not actually used in runtime.
//
// The value 2^30 is the max value AFAIK to make Go compiler happy on all archs.
const maxByteSize = 1 << 30

// hasPointers reports whether memory of type t can hold pointers.
//
// Memory that holds pointers must be written with typed stores so that
// the GC write barriers run. Memory without pointers can be cleared or
// copied as raw bytes, which is faster and doesn't allocate.
func hasPointers(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128:
		return false

	case reflect.Array:
		return t.Len() > 0 && hasPointers(t.Elem())

	case reflect.Struct:
		num := t.NumField()

		for i := 0; i < num; i++ {
			if hasPointers(t.Field(i).Type) {
				return true
			}
		}

		return false
	}

	return true
}

// clearBytes sets n bytes starting at p to zero.
// The memory must not hold pointers; see hasPointers.
//
// Pointers advance only while bytes remain, so they never point past the end
// of the allocation. checkptr rejects such pointers.
func clearBytes(p unsafe.Pointer, n uintptr) {
	clearBytesInChunks(p, n, maxByteSize)
}

// clearBytesInChunks is clearBytes with a configurable chunk size, so tests
// can cover chunk boundaries without gigabyte buffers.
// chunkSize must be in (0, maxByteSize]: 0 loops forever and a larger value
// overflows the slice bounds.
func clearBytesInChunks(p unsafe.Pointer, n, chunkSize uintptr) {
	for n > chunkSize {
		clearChunk(p, chunkSize)
		n -= chunkSize
		p = unsafe.Pointer(uintptr(p) + chunkSize)
	}

	if n > 0 {
		clearChunk(p, n)
	}
}

func clearChunk(p unsafe.Pointer, n uintptr) {
	b := (*[maxByteSize]byte)(p)[:n:n]

	// The compiler turns this loop into a single memclr call.
	for i := range b {
		b[i] = 0
	}
}

// copyBytes copies n bytes from src to dst.
// The memory must not hold pointers; see hasPointers.
func copyBytes(dst, src unsafe.Pointer, n uintptr) {
	copyBytesInChunks(dst, src, n, maxByteSize)
}

// copyBytesInChunks is copyBytes with a configurable chunk size.
// chunkSize must be in (0, maxByteSize]; see clearBytesInChunks.
func copyBytesInChunks(dst, src unsafe.Pointer, n, chunkSize uintptr) {
	for n > chunkSize {
		copy((*[maxByteSize]byte)(dst)[:chunkSize:chunkSize], (*[maxByteSize]byte)(src)[:chunkSize:chunkSize])
		n -= chunkSize
		dst = unsafe.Pointer(uintptr(dst) + chunkSize)
		src = unsafe.Pointer(uintptr(src) + chunkSize)
	}

	if n > 0 {
		copy((*[maxByteSize]byte)(dst)[:n:n], (*[maxByteSize]byte)(src)[:n:n])
	}
}
