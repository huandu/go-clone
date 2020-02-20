package clone

import (
	"fmt"
	"os"
	"reflect"
)

type ScalarType struct {
	stderr *os.File
}

func ExampleMarkAsScalar() {
	MarkAsScalar(reflect.TypeOf(new(ScalarType)))

	scalar := &ScalarType{
		stderr: os.Stderr,
	}
	cloned := Clone(scalar).(*ScalarType)

	// cloned is a shadow copy of scalar
	// so that the pointer value should be the same.
	fmt.Println(scalar.stderr == cloned.stderr)

	// Output:
	// true
}
