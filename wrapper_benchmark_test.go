// Copyright 2019 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

package clone

import "testing"

func BenchmarkUnwrap(b *testing.B) {
	orig := &testType{
		Foo: "abcd",
		Bar: map[string]interface{}{
			"def": 123,
			"ghi": 78.9,
		},
		Player: []float64{
			12.3, 45.6, -78.9,
		},
	}
	wrapped := Wrap(orig)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		Unwrap(wrapped)
	}
}

func BenchmarkSimpleWrap(b *testing.B) {
	orig := &testSimple{
		Foo: 123,
		Bar: "abcd",
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		Wrap(orig)
	}
}

func BenchmarkComplexWrap(b *testing.B) {
	orig := &testType{
		Foo: "abcd",
		Bar: map[string]interface{}{
			"def": 123,
			"ghi": 78.9,
		},
		Player: []float64{
			12.3, 45.6, -78.9,
		},
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		Wrap(orig)
	}
}
