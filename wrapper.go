// Copyright 2019 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

package clone

import (
	"reflect"
	"runtime"
	"sync"
	"unsafe"
)

var (
	// cachedWrapperTypes caches the internal wrapper type of a wrapped type.
	cachedWrapperTypes sync.Map

	// wrapped records every live wrapper allocation, keyed by its address.
	//
	// The key is a uintptr rather than a pointer on purpose: an address does not
	// keep the wrapper alive, so wrapped values stay collectable and the
	// finalizer installed in Wrap can drop the entry afterwards. Only equality
	// is ever asked of the key, so no pointer is reconstructed from it.
	wrapped sync.Map

	// wrappedOriginals maps a wrapper address to the original pointer it was
	// created from, as a reflect.Value of that exact pointer type. Holding the
	// origin as a reflect.Value lets Unwrap return it without rebuilding a
	// pointer out of a uintptr.
	wrappedOriginals sync.Map
)

// Wrap creates a wrapper of v, which must be a pointer.
// If v is not a pointer, Wrap simply returns v and do nothing.
//
// The wrapper is a deep clone of v's value. It holds a shadow copy to v internally.
//
//	t := &T{Foo: 123}
//	v := Wrap(t).(*T)               // v is a clone of t.
//	reflect.DeepEqual(t, v) == true // v equals t.
//	v.Foo = 456                     // v.Foo is changed, but t.Foo doesn't change.
//	orig := Unwrap(v)               // Use `Unwrap` to discard wrapper and return original value, which is t.
//	orig.(*T) == t                  // orig and t is exactly the same.
//	Undo(v)                         // Use `Undo` to discard any change on v.
//	v.Foo == t.Foo                  // Now, the value of v and t are the same again.
func Wrap(v interface{}) interface{} {
	if v == nil {
		return v
	}

	val := reflect.ValueOf(v)
	pt := val.Type()

	if val.Kind() != reflect.Ptr {
		return v
	}

	elem := val.Elem()
	ptr := unsafe.Pointer(val.Pointer())

	wrapperType := wrapperTypeOf(pt)
	pw := defaultAllocator.New(wrapperType)

	wrapperPtr := unsafe.Pointer(pw.Pointer())
	wrapper := pw.Elem()

	// Equivalent code: wrapper.T = Clone(v)
	field := wrapper.Field(0)
	field.Set(heapCloneState.clone(elem))

	// Equivalent code: wrapper.Origin = v
	wrapper.Field(1).SetPointer(ptr)

	// Register the wrapper allocation. This registry is what Unwrap/Undo consult
	// to tell a wrapped value from an arbitrary pointer. It replaces the old
	// probe that read the bytes following the wrapped value, which lay outside
	// the allocation for every pointer Wrap had not produced.
	//
	// val itself is the canonical reflect.Value of the original pointer, so it is
	// stored as is. Rebuilding the pointer later from the recorded address would
	// require a uintptr -> unsafe.Pointer conversion, which is exactly the unsafe
	// idiom this fix removes.
	addr := uintptr(wrapperPtr)
	wrapped.Store(addr, struct{}{})
	wrappedOriginals.Store(addr, val)

	// Drop the registry entry once the wrapper becomes unreachable, so these maps
	// do not grow without bound in long running programs. The finalizer closes
	// over addr, a plain integer, so it never keeps the wrapper alive.
	runtime.SetFinalizer(pw.Interface(), func(w interface{}) {
		wrapped.Delete(addr)
		wrappedOriginals.Delete(addr)
	})
	runtime.KeepAlive(pw)

	return field.Addr().Interface()
}

// wrapperTypeOf returns the internal wrapper type of the pointer type pt.
//
// The wrapper is
//
//	struct {
//	    T      <elem of pt>   // embedded, always at offset 0
//	    Origin unsafe.Pointer
//	}
//
// so the pointer Wrap returns is exactly a *T pointing at the start of this
// larger allocation.
//
// There is deliberately no Checksum field. It used to hold a crc64 of the
// wrapper and original addresses so that a pointer could be recognized by
// reading the bytes right after the wrapped value; that read is the checkptr
// violation this file now avoids. Recognition is done with the wrapped
// registry below, which is authoritative, so the checksum would never be
// consulted and only cost a crc64 computation per Wrap.
func wrapperTypeOf(pt reflect.Type) reflect.Type {
	if cache, ok := cachedWrapperTypes.Load(pt); ok {
		return cache.(reflect.Type)
	}

	created := reflect.StructOf([]reflect.StructField{
		{
			Name:      "T",
			Type:      pt.Elem(),
			Anonymous: true,
		},
		{
			Name: "Origin",
			Type: reflect.TypeOf(unsafe.Pointer(nil)),
		},
	})

	actual, _ := cachedWrapperTypes.LoadOrStore(pt, created)
	return actual.(reflect.Type)
}

// isWrapped reports whether val is a pointer returned by Wrap.
//
// Detection is a plain map lookup, so an arbitrary pointer never has the memory
// around it dereferenced. The previous implementation computed ptr+sizeof(T)+8
// to reach the wrapper metadata, which lies outside the allocation for any *T
// that Wrap did not produce; the checkptr instrumentation enabled by
// `go test -race` (and by -gcflags=-d=checkptr) turns that into a fatal,
// unrecoverable error.
func isWrapped(val reflect.Value) bool {
	if !val.IsValid() || val.Kind() != reflect.Ptr {
		return false
	}

	if val.Pointer() == 0 {
		return false
	}

	_, ok := wrapped.Load(uintptr(val.Pointer()))
	return ok
}

// originOf returns the original pointer stored for a wrapped value.
// The caller must have established that val is a wrapped value.
func originOf(val reflect.Value) reflect.Value {
	addr := uintptr(val.Pointer())
	orig, ok := wrappedOriginals.Load(addr)
	if !ok {
		return reflect.Value{}
	}

	return orig.(reflect.Value)
}

// Unwrap returns v's original value if v is a wrapped value.
// Otherwise, simply returns v itself.
func Unwrap(v interface{}) interface{} {
	if v == nil {
		return v
	}

	val := reflect.ValueOf(v)

	if !isWrapped(val) {
		return v
	}

	origVal := originOf(val)
	return origVal.Interface()
}

// Undo discards any change made in wrapped value.
// If v is not a wrapped value, nothing happens.
func Undo(v interface{}) {
	if v == nil {
		return
	}

	val := reflect.ValueOf(v)

	if !isWrapped(val) {
		return
	}

	origVal := originOf(val)
	elem := val.Elem()
	elem.Set(heapCloneState.clone(origVal.Elem()))
}
