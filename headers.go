package clone

import "reflect"

// As golint reports warning on possible misuse of these headers,
// avoid to use these header types directly to silience golint.

type sliceHeader reflect.SliceHeader
type stringHeader reflect.StringHeader
