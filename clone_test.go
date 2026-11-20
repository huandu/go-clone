// Copyright 2023 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

package clone

import (
	"runtime"
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

func TestClonePrivateFieldsInMapValues(t *testing.T) {
	type entry struct {
		value  *int
		alias  interface{}
		labels []string
	}
	n := 42
	src := map[int]entry{1: {value: &n, alias: &n, labels: []string{"first"}}}
	dst := Slowly(src).(map[int]entry)[1]
	if dst.value == &n || *dst.value != n || dst.alias != dst.value {
		t.Fatal("private pointer and interface must refer to the same cloned value")
	}
	dst.labels[0] = "second"
	if src[1].labels[0] != "first" {
		t.Fatal("cloned slice shares its backing array with the source")
	}
}

func TestSkipTagWithOnlyShallowFields(t *testing.T) {
	type record struct {
		shared  *int `clone:"shadowcopy"`
		skipped *int `clone:"-"`
	}
	n := 42
	src := struct{ entry record }{record{shared: &n, skipped: &n}}
	dst := Slowly(src).(struct{ entry record })
	if dst.entry.shared != &n || dst.entry.skipped != nil {
		t.Fatalf("unexpected tagged fields after clone: %+v", dst.entry)
	}
}

// TestCloneSurvivesGCWithPrivatePointers checks that pointers stored into
// unexported fields retain their values and aliasing across a GC cycle.
func TestCloneSurvivesGCWithPrivatePointers(t *testing.T) {
	type node struct {
		val  *int
		tags map[string]*int
	}
	type holder struct {
		nodes []node
	}

	const rounds = 200
	var srcs []holder
	for i := 0; i < 64; i++ {
		v := i
		srcs = append(srcs, holder{nodes: []node{{val: &v, tags: map[string]*int{"v": &v}}}})
	}

	const roundsPerGC = 20
	retained := make([]holder, 0, roundsPerGC*len(srcs))
	for r := 1; r <= rounds; r++ {
		for _, src := range srcs {
			retained = append(retained, Slowly(src).(holder))
		}
		if r%roundsPerGC != 0 && r != rounds {
			continue
		}

		runtime.GC()
		for j, got := range retained {
			i := j % len(srcs)
			n := got.nodes[0]
			if n.val == nil || *n.val != i {
				t.Fatalf("round %d, clone %d: lost value after GC", r, j)
			}
			if n.tags["v"] != n.val {
				t.Fatalf("round %d, clone %d: lost internal alias after GC", r, j)
			}
			if n.val == srcs[i].nodes[0].val {
				t.Fatalf("round %d, clone %d: shares pointer with source", r, j)
			}
		}
		retained = retained[:0]
	}
}
