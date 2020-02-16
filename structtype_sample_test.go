package clone

import (
	"fmt"
	"os"
	"reflect"
)

type ScalaType struct {
	stderr *os.File
}

func ExampleMarkAsScala() {
	MarkAsScala(reflect.TypeOf(new(ScalaType)))

	scala := &ScalaType{
		stderr: os.Stderr,
	}
	cloned := Clone(scala).(*ScalaType)

	// cloned is a shadow copy of scala
	// so that the pointer value should be the same.
	fmt.Println(scala.stderr == cloned.stderr)

	// Output:
	// true
}
