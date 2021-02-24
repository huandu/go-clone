// Copyright 2019 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

package clone

import "testing"

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
