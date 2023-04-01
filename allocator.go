// Copyright 2023 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

package clone

import (
	"reflect"
	"runtime"
	"sync"
	"unsafe"
)

var typeOfAllocator = reflect.TypeOf(Allocator{})

// The heapAllocator allocates memory from heap.
var heapAllocator = FromHeap()

// Allocator is a utility type for memory allocation.
type Allocator struct {
	pool      unsafe.Pointer
	new       func(pool unsafe.Pointer, t reflect.Type) reflect.Value
	makeSlice func(pool unsafe.Pointer, t reflect.Type, len, cap int) reflect.Value
	makeMap   func(pool unsafe.Pointer, t reflect.Type, n int) reflect.Value
	makeChan  func(pool unsafe.Pointer, t reflect.Type, buffer int) reflect.Value
	isScalar  func(t reflect.Kind) bool

	cachedStructTypes     sync.Map
	cachedPointerTypes    sync.Map
	cachedCustomFuncTypes sync.Map
}

// AllocatorMethods defines all methods required by allocator.
// If any of these methods is nil, allocator will use default method which allocates memory from heap.
type AllocatorMethods struct {
	New       func(pool unsafe.Pointer, t reflect.Type) reflect.Value
	MakeSlice func(pool unsafe.Pointer, t reflect.Type, len, cap int) reflect.Value
	MakeMap   func(pool unsafe.Pointer, t reflect.Type, n int) reflect.Value
	MakeChan  func(pool unsafe.Pointer, t reflect.Type, buffer int) reflect.Value
	IsScalar  func(t reflect.Kind) bool
}

// FromHeap creates an allocator which allocate memory from heap.
func FromHeap() *Allocator {
	return NewAllocator(nil, nil)
}

// NewAllocator creates an allocator which allocate memory from the pool.
// Both pool and methods are optional.
//
// If methods.New is not nil, the allocator itself is created by calling methods.New.
//
// The pool is a pointer to the memory pool which is opaque to the allocator.
// It's methods's responsibility to allocate memory from the pool properly.
func NewAllocator(pool unsafe.Pointer, methods *AllocatorMethods) (allocator *Allocator) {
	if methods == nil {
		allocator = &Allocator{
			pool: pool,

			cachedStructTypes:     sync.Map{},
			cachedPointerTypes:    sync.Map{},
			cachedCustomFuncTypes: sync.Map{},
		}
	} else if methods.New == nil {
		allocator = &Allocator{
			pool:      pool,
			new:       methods.New,
			makeSlice: methods.MakeSlice,
			makeMap:   methods.MakeMap,
			makeChan:  methods.MakeChan,
			isScalar:  methods.IsScalar,

			cachedStructTypes:     sync.Map{},
			cachedPointerTypes:    sync.Map{},
			cachedCustomFuncTypes: sync.Map{},
		}
	} else {
		// Allocate the allocator from the pool.
		val := methods.New(pool, typeOfAllocator)
		allocator = (*Allocator)(unsafe.Pointer(val.Pointer()))
		runtime.KeepAlive(val)

		*allocator = Allocator{
			pool:      pool,
			new:       methods.New,
			makeSlice: methods.MakeSlice,
			makeMap:   methods.MakeMap,
			makeChan:  methods.MakeChan,
			isScalar:  methods.IsScalar,

			cachedStructTypes:     sync.Map{},
			cachedPointerTypes:    sync.Map{},
			cachedCustomFuncTypes: sync.Map{},
		}
	}

	if allocator.new == nil {
		allocator.new = heapNew
	}

	if allocator.makeSlice == nil {
		allocator.makeSlice = heapMakeSlice
	}

	if allocator.makeMap == nil {
		allocator.makeMap = heapMakeMap
	}

	if allocator.makeChan == nil {
		allocator.makeChan = heapMakeChan
	}

	if allocator.isScalar == nil {
		allocator.isScalar = IsScalar
	}

	return allocator
}

// New returns a new zero value of t.
func (a *Allocator) New(t reflect.Type) reflect.Value {
	return a.new(a.pool, t)
}

// MakeSlice creates a new zero-initialized slice value of t with len and cap.
func (a *Allocator) MakeSlice(t reflect.Type, len, cap int) reflect.Value {
	return a.makeSlice(a.pool, t, len, cap)
}

// MakeMap creates a new map with minimum size n.
func (a *Allocator) MakeMap(t reflect.Type, n int) reflect.Value {
	return a.makeMap(a.pool, t, n)
}

// MakeChan creates a new chan with buffer.
func (a *Allocator) MakeChan(t reflect.Type, buffer int) reflect.Value {
	return a.makeChan(a.pool, t, buffer)
}

// Clone recursively deep clone val to a new value with memory allocated from a.
func (a *Allocator) Clone(val reflect.Value) reflect.Value {
	return a.clone(val, true)
}

func (a *Allocator) clone(val reflect.Value, inCustomFunc bool) reflect.Value {
	if !val.IsValid() {
		return val
	}

	state := &cloneState{
		allocator: a,
	}

	if inCustomFunc {
		state.skipCustomFuncValue = val
	}

	return state.clone(val)
}

// CloneSlowly recursively deep clone val to a new value with memory allocated from a.
// It marks all cloned values internally, thus it can clone v with cycle pointer.
func (a *Allocator) CloneSlowly(val reflect.Value) reflect.Value {
	return a.cloneSlowly(val, true)
}

func (a *Allocator) cloneSlowly(val reflect.Value, inCustomFunc bool) reflect.Value {
	if !val.IsValid() {
		return val
	}

	state := &cloneState{
		allocator: a,
		visited:   visitMap{},
		invalid:   invalidPointers{},
	}

	if inCustomFunc {
		state.skipCustomFuncValue = val
	}

	cloned := state.clone(val)
	state.fix(cloned)
	return cloned
}

func (a *Allocator) loadStructType(t reflect.Type) (st structType) {
	if v, ok := a.cachedStructTypes.Load(t); ok {
		st = v.(structType)
		return
	}

	num := t.NumField()
	pointerFields := make([]structFieldType, 0, num)

	for i := 0; i < num; i++ {
		field := t.Field(i)
		ft := field.Type
		k := ft.Kind()

		if a.isScalar(k) {
			continue
		}

		switch k {
		case reflect.Array:
			if ft.Len() == 0 {
				continue
			}

			elem := ft.Elem()

			if a.isScalar(elem.Kind()) {
				continue
			}

			if elem.Kind() == reflect.Struct {
				if fst := a.loadStructType(elem); fst.CanShadowCopy() {
					continue
				}
			}
		case reflect.Struct:
			if fst := a.loadStructType(ft); fst.CanShadowCopy() {
				continue
			}
		}

		pointerFields = append(pointerFields, structFieldType{
			Offset: field.Offset,
			Index:  i,
		})
	}

	if len(pointerFields) == 0 {
		pointerFields = nil // Release memory ASAP.
	}

	st = structType{
		PointerFields: pointerFields,
	}

	if fn, ok := a.cachedCustomFuncTypes.Load(t); ok {
		st.fn = fn.(Func)
	}

	a.cachedStructTypes.LoadOrStore(t, st)
	return
}

func (a *Allocator) isOpaquePointer(t reflect.Type) (ok bool) {
	_, ok = a.cachedPointerTypes.Load(t)
	return
}

// MarkAsScalar marks t as a scalar type so that all clone methods will copy t by value.
// If t is not struct or pointer to struct, MarkAsScalar ignores t.
//
// In the most cases, it's not necessary to call it explicitly.
// If a struct type contains scalar type fields only, the struct will be marked as scalar automatically.
//
// Here is a list of types marked as scalar by default:
//   - time.Time
//   - reflect.Value
func (a *Allocator) MarkAsScalar(t reflect.Type) {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return
	}

	a.cachedStructTypes.Store(t, zeroStructType)
}

// MarkAsOpaquePointer marks t as an opaque pointer so that all clone methods will copy t by value.
// If t is not a pointer, MarkAsOpaquePointer ignores t.
//
// Here is a list of types marked as opaque pointers by default:
//   - `elliptic.Curve`, which is `*elliptic.CurveParam` or `elliptic.p256Curve`;
//   - `reflect.Type`, which is `*reflect.rtype` defined in `runtime`.
func (a *Allocator) MarkAsOpaquePointer(t reflect.Type) {
	if t.Kind() != reflect.Ptr {
		return
	}

	a.cachedPointerTypes.Store(t, struct{}{})
}

// SetCustomFunc sets a custom clone function for type t.
// If t is not struct or pointer to struct, SetCustomFunc ignores t.
//
// If fn is nil, remove the custom clone function for type t.
func (a *Allocator) SetCustomFunc(t reflect.Type, fn Func) {
	if fn == nil {
		a.cachedCustomFuncTypes.Delete(t)
		return
	}

	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return
	}

	a.cachedCustomFuncTypes.Store(t, fn)
}

func heapNew(pool unsafe.Pointer, t reflect.Type) reflect.Value {
	return reflect.New(t)
}

func heapMakeSlice(pool unsafe.Pointer, t reflect.Type, len, cap int) reflect.Value {
	return reflect.MakeSlice(t, len, cap)
}

func heapMakeMap(pool unsafe.Pointer, t reflect.Type, n int) reflect.Value {
	return reflect.MakeMapWithSize(t, n)
}

func heapMakeChan(pool unsafe.Pointer, t reflect.Type, buffer int) reflect.Value {
	return reflect.MakeChan(t, buffer)
}
