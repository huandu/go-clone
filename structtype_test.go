package clone

import (
	"reflect"
	"testing"

	"github.com/huandu/go-assert"
)

type NoPointer struct {
	Foo int
	Bar string
}

type WithPointer struct {
	foo map[string]string
	bar []int
}

func TestMarkAsScala(t *testing.T) {
	a := assert.New(t)
	oldCnt := 0
	newCnt := 0
	a.Use(&oldCnt, &newCnt)

	// Count cache.
	cachedStructTypes.Range(func(key, value interface{}) bool {
		oldCnt++
		return true
	})

	// Add 2 valid types.
	MarkAsScala(reflect.TypeOf(new(NoPointer)))
	MarkAsScala(reflect.TypeOf(new(WithPointer)))
	MarkAsScala(reflect.TypeOf(new(int))) // Should be ignored.

	// Count cache against.
	cachedStructTypes.Range(func(key, value interface{}) bool {
		newCnt++
		return true
	})

	a.Assert(oldCnt+2 == newCnt)

	// As WithPointer is marked as scala, Clone returns a shadow copy.
	value := &WithPointer{
		foo: map[string]string{
			"key": "value",
		},
		bar: []int{1, 2, 3},
	}
	cloned := Clone(value).(*WithPointer)
	a.Use(&value, &cloned)

	// cloned is a shadow copy.
	a.Equal(value, cloned)
	value.foo["key"] = "modified"
	value.bar[1] = 2000
	a.Equal(value, cloned)
}
