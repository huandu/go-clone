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
