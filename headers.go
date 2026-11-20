// Copyright 2019 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

package clone

import "unsafe"

// stringHeader mirrors the runtime layout of a string.
// Data is a pointer, not an uintptr, so assigning a stringHeader is a typed
// store that runs the GC write barrier.
type stringHeader struct {
	Data unsafe.Pointer
	Len  int
}
