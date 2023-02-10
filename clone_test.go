// Copyright 2023 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

package clone

import "testing"

func TestCloneAll(t *testing.T) {
	for name, fn := range testFuncMap {
		t.Run(name, func(t *testing.T) {
			fn(t, heapAllocator)
		})
	}
}
