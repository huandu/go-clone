// Copyright 2019 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

package clone

import (
	"crypto/elliptic"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/huandu/go-assert"
)

type NoPointer struct {
	Foo int
	Bar string
}

type WithPointer struct {
	foo map[string]string
	bar []int
}

func TestMarkAsScalar(t *testing.T) {
	a := assert.New(t)
	oldCnt := 0
	newCnt := 0
	a.Use(&oldCnt, &newCnt)

	// Count cache.
	defaultAllocator.cachedStructTypes.Range(func(key, value interface{}) bool {
		oldCnt++
		return true
	})

	// Add 2 valid types.
	MarkAsScalar(reflect.TypeOf(new(NoPointer)))
	MarkAsScalar(reflect.TypeOf(new(WithPointer)))
	MarkAsScalar(reflect.TypeOf(new(int))) // Should be ignored.

	// Count cache against.
	defaultAllocator.cachedStructTypes.Range(func(key, value interface{}) bool {
		newCnt++
		return true
	})

	a.Assert(oldCnt+2 == newCnt)

	// As WithPointer is marked as scalar, Clone returns a shadow copy.
	value := &WithPointer{
		foo: map[string]string{
			"key": "value",
		},
		bar: []int{1, 2, 3},
	}
	cloned := Clone(value).(*WithPointer)
	a.Use(&value, &cloned)

	// cloned is a shadow copy.
	a.Equal(value, cloned)
	value.foo["key"] = "modified"
	value.bar[1] = 2000
	a.Equal(value, cloned)
}

type MapKeys struct {
	mb       map[bool]interface{}
	mi       map[int]interface{}
	mi8      map[int8]interface{}
	mi16     map[int16]interface{}
	mi32     map[int32]interface{}
	mi64     map[int64]interface{}
	mui      map[uint]interface{}
	mu8      map[uint8]interface{}
	mu16     map[uint16]interface{}
	mu32     map[uint32]interface{}
	mu64     map[uint64]interface{}
	muintptr map[uintptr]interface{}
	ms       map[string]interface{}
	mf32     map[float32]interface{}
	mf64     map[float64]interface{}
	mc64     map[complex64]interface{}
	mc128    map[complex128]interface{}
	miface   map[interface{}]interface{}
	mis      map[Simple]interface{}
	// misp     map[*Simple]interface{}
	munsafe map[unsafe.Pointer]interface{}
}

func TestCopyScalarValue(t *testing.T) {
	a := assert.New(t)
	st := &MapKeys{
		mb:       map[bool]interface{}{true: 2},
		mi:       map[int]interface{}{-1: 2},
		mi8:      map[int8]interface{}{-8: 2},
		mi16:     map[int16]interface{}{-16: 2},
		mi32:     map[int32]interface{}{-32: 2},
		mi64:     map[int64]interface{}{-64: 2},
		mui:      map[uint]interface{}{1: 2},
		mu8:      map[uint8]interface{}{8: 2},
		mu16:     map[uint16]interface{}{16: 2},
		mu32:     map[uint32]interface{}{32: 2},
		mu64:     map[uint64]interface{}{64: 2},
		muintptr: map[uintptr]interface{}{0xDEADC0DE: 2},
		ms:       map[string]interface{}{"str": 2},
		mf32:     map[float32]interface{}{3.2: 2},
		mf64:     map[float64]interface{}{6.4: 2},
		mc64:     map[complex64]interface{}{complex(6, 4): 2},
		mc128:    map[complex128]interface{}{complex(1.2, 8): 2},
		miface:   map[interface{}]interface{}{"iface": 2},
		mis:      map[Simple]interface{}{{Foo: 123}: 2},
		munsafe:  map[unsafe.Pointer]interface{}{unsafe.Pointer(t): 2},
	}
	cloned := Clone(st).(*MapKeys)

	a.Equal(st, cloned)
}

type noCopyValues struct {
	syncMutex     sync.Mutex
	syncRWMutex   sync.RWMutex
	syncWaitGroup sync.WaitGroup
	syncCond      *sync.Cond
	syncPool      sync.Pool
	syncMap       sync.Map
	syncOnce      sync.Once
	atomicValue   atomic.Value
}

func TestCloneNoCopyValues(t *testing.T) {
	a := assert.New(t)
	v := &noCopyValues{
		syncCond: sync.NewCond(func() *sync.Mutex {
			return &sync.Mutex{}
		}()),
		syncPool: sync.Pool{
			New: func() interface{} {
				return "pool"
			},
		},
	}

	v.syncMutex.Lock()
	defer v.syncMutex.Unlock()
	v.syncRWMutex.RLock()
	defer v.syncRWMutex.RUnlock()
	v.syncWaitGroup.Add(1)
	defer v.syncWaitGroup.Done()
	v.syncCond.L.Lock()
	defer v.syncCond.L.Unlock()
	poolValue := v.syncPool.Get()
	v.syncPool.Put(poolValue)
	v.syncMap.Store("foo", "bar")
	v.syncOnce.Do(func() {})
	v.atomicValue.Store("value")

	cloned := Clone(v).(*noCopyValues)
	done := make(chan bool, 1)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	a.Run("race", func(t *testing.T) {
		a := assert.New(t)

		cloned.syncMutex.Lock()
		_ = 0
		cloned.syncMutex.Unlock()

		cloned.syncRWMutex.RLock()
		_ = 0
		cloned.syncRWMutex.RUnlock()

		cloned.syncWaitGroup.Add(1)
		cloned.syncWaitGroup.Done()
		cloned.syncWaitGroup.Wait()

		cloned.syncCond.L.Lock()
		_ = 0
		cloned.syncCond.L.Unlock()

		poolValue := cloned.syncPool.Get()
		a.Equal(poolValue, "pool")

		mapValue, ok := cloned.syncMap.Load("foo")
		a.Equal(mapValue, "bar")
		a.Assert(ok)

		onceValueShouldBeTrue := true
		cloned.syncOnce.Do(func() {
			onceValueShouldBeTrue = false
		})
		a.Assert(onceValueShouldBeTrue)

		value := cloned.atomicValue.Load()
		a.Equal(value, "value")

		done <- true
	})

	select {
	case <-done:
	case <-ticker.C:
		a.Fatalf("unexpected lock is detected.")
	}
}

func TestCloneCurveAsScalar(t *testing.T) {
	a := assert.New(t)
	curves := []elliptic.Curve{elliptic.P224(), elliptic.P256(), elliptic.P384(), elliptic.P521()}
	cloned := Clone(curves).([]elliptic.Curve)

	for i, curve := range curves {
		c := cloned[i]
		a.Assert(curve == c)
	}
}

type testOpaquePointer struct {
	foo int
}

func TestMarkAsOpaquePointer(t *testing.T) {
	a := assert.New(t)

	// Mark *testOpaquePointer as opaque pointer.
	MarkAsOpaquePointer(reflect.TypeOf(&testOpaquePointer{}))

	// No-op if set a struct type as opaque.
	MarkAsOpaquePointer(reflect.TypeOf(testOpaquePointer{}))

	opaque := &testOpaquePointer{
		foo: 1234,
	}
	cloned := Clone(&opaque).(**testOpaquePointer)

	a.Assert(&opaque != cloned)
	a.Assert(opaque == *cloned)
}
