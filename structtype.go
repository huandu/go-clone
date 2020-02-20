package clone

import (
	"fmt"
	"reflect"
	"sync"
	"time"
	"unsafe"
)

var (
	cachedStructTypes sync.Map
)

func init() {
	// Some well-known scalar-like structs.
	MarkAsScalar(reflect.TypeOf(time.Time{}))
	MarkAsScalar(reflect.TypeOf(reflect.Value{}))
}

// MarkAsScalar marks t as a scalar type so that all clone methods will copy t by value.
// If t is not struct or pointer to struct, MarkAsScalar ignores t.
//
// In the most cases, it's not necessary to call it explicitly.
// If a struct type contains scalar type fields only, the struct will be marked as scalar automatically.
//
// Here is a list of types marked as scalar by default:
//     * time.Time
//     * reflect.Value
func MarkAsScalar(t reflect.Type) {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return
	}

	cachedStructTypes.Store(t, structType{})
}

type structType struct {
	PointerFields []structFieldType
}

type structFieldType struct {
	Offset uintptr // The offset from the beginning of the struct.
	Index  int     // The index of the field.
}

func loadStructType(t reflect.Type) (st structType) {
	if v, ok := cachedStructTypes.Load(t); ok {
		st = v.(structType)
		return
	}

	num := t.NumField()
	pointerFields := make([]structFieldType, 0, num)

	for i := 0; i < num; i++ {
		field := t.Field(i)
		ft := field.Type
		k := ft.Kind()

		if isScalar(k) {
			continue
		}

		switch k {
		case reflect.Array:
			if ft.Len() == 0 {
				continue
			}

			elem := ft.Elem()

			if isScalar(elem.Kind()) {
				continue
			}

			if elem.Kind() == reflect.Struct {
				fst := loadStructType(elem)

				if len(fst.PointerFields) == 0 {
					continue
				}
			}
		case reflect.Struct:
			fst := loadStructType(ft)

			if len(fst.PointerFields) == 0 {
				continue
			}
		}

		pointerFields = append(pointerFields, structFieldType{
			Offset: field.Offset,
			Index:  i,
		})
	}

	if len(pointerFields) == 0 {
		pointerFields = nil // Release memory ASAP.
	}

	st = structType{
		PointerFields: pointerFields,
	}
	cachedStructTypes.LoadOrStore(t, st)
	return
}

func isScalar(k reflect.Kind) bool {
	switch k {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128,
		reflect.String, reflect.Func,
		reflect.UnsafePointer,
		reflect.Invalid:
		return true
	}

	return false
}

type baitType struct{}

func (baitType) Foo() {}

var (
	baitMethodValue = reflect.ValueOf(baitType{}).MethodByName("Foo")
)

func copyScalarValue(src reflect.Value) reflect.Value {
	if src.CanInterface() {
		return src
	}

	// src is an unexported field value. Copy its value.
	switch src.Kind() {
	case reflect.Bool:
		return reflect.ValueOf(src.Bool())

	case reflect.Int:
		return reflect.ValueOf(int(src.Int()))
	case reflect.Int8:
		return reflect.ValueOf(int8(src.Int()))
	case reflect.Int16:
		return reflect.ValueOf(int16(src.Int()))
	case reflect.Int32:
		return reflect.ValueOf(int32(src.Int()))
	case reflect.Int64:
		return reflect.ValueOf(src.Int())

	case reflect.Uint:
		return reflect.ValueOf(uint(src.Uint()))
	case reflect.Uint8:
		return reflect.ValueOf(uint8(src.Uint()))
	case reflect.Uint16:
		return reflect.ValueOf(uint16(src.Uint()))
	case reflect.Uint32:
		return reflect.ValueOf(uint32(src.Uint()))
	case reflect.Uint64:
		return reflect.ValueOf(src.Uint())
	case reflect.Uintptr:
		return reflect.ValueOf(uintptr(src.Uint()))

	case reflect.Float32:
		return reflect.ValueOf(float32(src.Float()))
	case reflect.Float64:
		return reflect.ValueOf(src.Float())

	case reflect.Complex64:
		return reflect.ValueOf(complex64(src.Complex()))
	case reflect.Complex128:
		return reflect.ValueOf(src.Complex())

	case reflect.String:
		return reflect.ValueOf(src.String())
	case reflect.Func:
		t := src.Type()

		if src.IsNil() {
			return reflect.Zero(t)
		}

		// Don't use this trick unless we have no choice.
		return forceClearROFlag(src)
	case reflect.UnsafePointer:
		return reflect.ValueOf(unsafe.Pointer(src.Pointer()))
	}

	panic(fmt.Errorf("go-clone: <bug> impossible type `%v` when cloning private field", src.Type()))
}

var typeOfInterface = reflect.TypeOf((*interface{})(nil)).Elem()

// forceClearROFlag clears all RO flags in v to make v accessible.
// It's a hack based on the fact that InterfaceData is always available on RO data.
// This hack can be broken in any Go version.
// Don't use it unless we have no choice, e.g. copying func in some edge cases.
func forceClearROFlag(v reflect.Value) reflect.Value {
	var i interface{}

	v = v.Convert(typeOfInterface)
	nv := reflect.ValueOf(&i)
	*(*[2]uintptr)(unsafe.Pointer(nv.Pointer())) = v.InterfaceData()
	return nv.Elem().Elem()
}
