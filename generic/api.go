// Copyright 2022 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

// Package clone provides functions to deep clone any Go data.
// It also provides a wrapper to protect a pointer from any unexpected mutation.
//
// This package is only a proxy to original go-clone package with generic support.
// To minimize the maintenace cost, there is no doc in this package.
// Please read the document in https://pkg.go.dev/github.com/huandu/go-clone instead.
package clone

import (
	"reflect"
	"unsafe"

	"github.com/huandu/go-clone"
)

type Func = clone.Func
type Allocator = clone.Allocator
type AllocatorMethods = clone.AllocatorMethods
type Cloner = clone.Cloner

func Clone[T any](t T) T {
	return clone.Clone(t).(T)
}

func Slowly[T any](t T) T {
	return clone.Slowly(t).(T)
}

func Wrap[T any](t T) T {
	return clone.Wrap(t).(T)
}

func Unwrap[T any](t T) T {
	return clone.Unwrap(t).(T)
}

func Undo[T any](t T) {
	clone.Undo(t)
}

func MarkAsOpaquePointer(t reflect.Type) {
	clone.MarkAsOpaquePointer(t)
}

func MarkAsScalar(t reflect.Type) {
	clone.MarkAsScalar(t)
}

func SetCustomFunc(t reflect.Type, fn Func) {
	clone.SetCustomFunc(t, fn)
}

func FromHeap() *Allocator {
	return clone.FromHeap()
}

func NewAllocator(pool unsafe.Pointer, methods *AllocatorMethods) (allocator *Allocator) {
	return clone.NewAllocator(pool, methods)
}

func IsScalar(k reflect.Kind) bool {
	return clone.IsScalar(k)
}

func MakeCloner(allocator *Allocator) Cloner {
	return clone.MakeCloner(allocator)
}
