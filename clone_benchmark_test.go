// Copyright 2019 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

package clone

import "testing"

// benchSink keeps benchmark results alive so the compiler can't drop the clone.
var benchSink interface{}

func BenchmarkSimpleClone(b *testing.B) {
	orig := &testSimple{
		Foo: 123,
		Bar: "abcd",
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		Clone(orig)
	}
}

func BenchmarkComplexClone(b *testing.B) {
	m := map[string]*T{
		"abc": {
			Foo: 123,
			Bar: map[string]interface{}{
				"abc": 321,
			},
		},
		"def": {
			Foo: 456,
			Bar: map[string]interface{}{
				"def": 789,
			},
		},
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		Clone(m)
	}
}

func BenchmarkClonePrivatePointers(b *testing.B) {
	type inner struct {
		p *int
		s []int
	}
	type outer struct {
		a inner
		b *inner
	}
	n := 1
	src := outer{a: inner{p: &n, s: []int{1, 2, 3}}, b: &inner{p: &n}}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		benchSink = Slowly(src)
	}
}

type benchPrivateSlices struct {
	ints []int
	strs []string
}

func newBenchPrivateSlices() *benchPrivateSlices {
	v := &benchPrivateSlices{
		ints: make([]int, 1024),
		strs: make([]string, 256),
	}

	for i := range v.ints {
		v.ints[i] = i
	}

	for i := range v.strs {
		v.strs[i] = "value"
	}

	return v
}

// BenchmarkClonePrivateSlices covers private scalar slices reached through a
// non-addressable value (struct passed by value) and an addressable one
// (struct behind a pointer).
func BenchmarkClonePrivateSlices(b *testing.B) {
	ptr := newBenchPrivateSlices()
	val := *ptr

	b.Run("value", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			benchSink = Clone(val)
		}
	})
	b.Run("pointer", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			benchSink = Clone(ptr)
		}
	})
}

type benchSkipFields struct {
	n     int
	keep  *int
	big   [4096]byte `clone:"-"`
	small [64]byte   `clone:"-"`
	ptr   *int       `clone:"-"`
}

// BenchmarkCloneSkipFields covers zeroing of skipped fields, including one
// larger than reflect's 1 KiB shared zero value.
func BenchmarkCloneSkipFields(b *testing.B) {
	n := 1
	ptr := &benchSkipFields{n: 1, keep: &n, ptr: &n}
	ptr.big[0], ptr.small[0] = 1, 1
	val := *ptr

	b.Run("value", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			benchSink = Clone(val)
		}
	})
	b.Run("pointer", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			benchSink = Clone(ptr)
		}
	})
}

// BenchmarkCloneMapPrivateValues covers map values, which reflect hands out
// as non-addressable copies, so their private fields take the slowest path.
func BenchmarkCloneMapPrivateValues(b *testing.B) {
	type entry struct {
		p *int
		s []int
	}
	n := 1
	m := make(map[int]entry, 64)

	for i := 0; i < 64; i++ {
		m[i] = entry{p: &n, s: []int{i, i + 1}}
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		benchSink = Clone(m)
	}
}
