// Copyright 2023 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

package clone

import (
	"sync/atomic"
	"testing"

	"github.com/huandu/go-assert"
)

func TestCloneAll(t *testing.T) {
	for name, fn := range testFuncMap {
		t.Run(name, func(t *testing.T) {
			fn(t, defaultAllocator)
		})
	}
}

// TestIssue21 tests issue #21.
func TestIssue21(t *testing.T) {
	a := assert.New(t)

	type Foo string
	type Bar struct {
		foo *Foo
	}

	foo := Foo("hello")

	src := Bar{
		foo: &foo,
	}

	dst := Clone(src).(Bar)

	a.Equal(dst.foo, src.foo)
	a.Assert(dst.foo != src.foo)
	a.Equal(dst, src)
}

// TestIssue25 tests issue #25.
func TestIssue25(t *testing.T) {
	a := assert.New(t)
	cloned := Clone(new(atomic.Value)).(*atomic.Value)

	a.Equal(cloned.Load(), nil)
}

func TestCloneDefinedPointer(t *testing.T) {
	type intPtr *int
	n := 42
	src := intPtr(&n)
	for name, clone := range map[string]func(interface{}) interface{}{"Clone": Clone, "Slowly": Slowly} {
		t.Run(name, func(t *testing.T) {
			got := clone(src)
			dst, ok := got.(intPtr)
			if !ok {
				t.Fatalf("cloned pointer has type %T, want %T", got, src)
			}
			if dst == nil || dst == src || *dst != n {
				t.Fatalf("want an independent pointer to %d, got %p", n, dst)
			}
		})
	}
}

func TestSlowlyDefinedPointerCycle(t *testing.T) {
	type nodePtr *struct{ next nodePtr }
	src := new(struct{ next nodePtr })
	src.next = src
	got := Slowly(nodePtr(src))
	dst, ok := got.(nodePtr)
	if !ok || dst == nil || dst == src || dst.next != dst {
		t.Fatalf("Slowly returned %T (%p), want a cloned %T pointing to itself", got, got, nodePtr(src))
	}
}
