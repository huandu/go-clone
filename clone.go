// Copyright 2019 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

// Package clone provides functions to deep clone any Go data.
// It also provides a wrapper to protect a pointer from any unexpected mutation.
package clone

import (
	"fmt"
	"math"
	"reflect"
	"unsafe"
)

// Clone recursively deep clone v to a new value.
// It assumes that there is no pointer cycle in v,
// e.g. v has a pointer points to v itself.
// If there is a pointer cycle, use Slowly instead.
//
// Clone cannot clone some special values due to Go's limitation.
// Here is the list of special values.
//
//     * chan: New empty chan is made without any data.
//     * func: Copied by value as func is immutable at runtime.
//     * unsafe.Pointer: Copied by pointer value as we don't know what's in it.
//
// Clone is able to clone unexported fields in a struct.
// It works fine in the most cases.
// The only known exception is the unexported method value.
// Following code shows the case.
//
//     type T struct {
//         fn func() int // fn is an unexported field.
//     }
//
//     t := &T{}
//     t.fn = func() int { return 123 }
//
//     // Clone can work fine with closure or func pointer.
//     cloned := Clone(t).(*T)
//     println(cloned.fn()) // 123
//
//     // However, Clone will clone an invalid fn pointer in this case.
//     // AFAIK, there is no way to fix it.
//     // Maybe the best way is to manually wrap cloned.fn in a func
//     // and handle panic in defer.
//     t.fn = bytes.NewBufferString("foo").Len
//     cloned = Clone(t).(*T)
//     print(cloned.fn == nil) // false
//     cloned.fn()          // panic
//
func Clone(v interface{}) interface{} {
	if v == nil {
		return v
	}

	val := reflect.ValueOf(v)
	cloned := clone(val, nil)
	return cloned.Interface()
}

// Slowly recursively deep clone v to a new value.
// It marks all cloned values internally, thus it can clone v with cycle pointer.
//
// Slowly works exactly the same as Clone. See Clone doc for more details.
func Slowly(v interface{}) interface{} {
	if v == nil {
		return v
	}

	val := reflect.ValueOf(v)
	cloned := clone(val, visitMap{})
	return cloned.Interface()
}

type visit struct {
	p     uintptr
	extra int
	t     reflect.Type
}

type visitMap map[visit]reflect.Value

func clone(v reflect.Value, visited visitMap) reflect.Value {
	if !v.IsValid() {
		return reflect.Value{}
	}

	if isScala(v.Kind()) {
		return copyScalaValue(v)
	}

	switch v.Kind() {
	case reflect.Array:
		return cloneArray(v, visited)
	case reflect.Chan:
		return reflect.MakeChan(v.Type(), v.Cap())
	case reflect.Interface:
		return cloneInterface(v, visited)
	case reflect.Map:
		return cloneMap(v, visited)
	case reflect.Ptr:
		return clonePtr(v, visited)
	case reflect.Slice:
		return cloneSlice(v, visited)
	case reflect.Struct:
		return cloneStruct(v, visited)
	default:
		panic(fmt.Errorf("go-clone: <bug> unsupported type `%v`", v.Type()))
	}
}

func cloneArray(v reflect.Value, visited visitMap) reflect.Value {
	dst := reflect.New(v.Type())
	copyArray(v, dst, visited)
	return dst.Elem()
}

func copyArray(src, dst reflect.Value, visited visitMap) {
	p := unsafe.Pointer(dst.Pointer()) // dst must be a Ptr.
	dst = dst.Elem()
	num := src.Len()

	if isScala(src.Type().Elem().Kind()) {
		shadowCopy(src, p)
		return
	}

	for i := 0; i < num; i++ {
		dst.Index(i).Set(clone(src.Index(i), visited))
	}
}

func cloneInterface(v reflect.Value, visited visitMap) reflect.Value {
	if v.IsNil() {
		return reflect.Zero(v.Type())
	}

	t := v.Type()
	return clone(v.Elem(), visited).Convert(t)
}

func cloneMap(v reflect.Value, visited visitMap) reflect.Value {
	if v.IsNil() {
		return reflect.Zero(v.Type())
	}

	t := v.Type()

	if visited != nil {
		visit := visit{
			p: v.Pointer(),
			t: t,
		}

		if val, ok := visited[visit]; ok {
			return val
		}
	}

	nv := reflect.MakeMap(t)

	if visited != nil {
		visit := visit{
			p: v.Pointer(),
			t: t,
		}
		visited[visit] = nv
	}

	for iter := mapIter(v); iter.Next(); {
		nv.SetMapIndex(clone(iter.Key(), visited), clone(iter.Value(), visited))
	}

	return nv
}

func clonePtr(v reflect.Value, visited visitMap) reflect.Value {
	if v.IsNil() {
		return reflect.Zero(v.Type())
	}

	t := v.Type()

	if visited != nil {
		visit := visit{
			p: v.Pointer(),
			t: t,
		}

		if val, ok := visited[visit]; ok {
			return val
		}
	}

	elemType := t.Elem()
	nv := reflect.New(elemType)

	if visited != nil {
		visit := visit{
			p: v.Pointer(),
			t: t,
		}
		visited[visit] = nv
	}

	p := unsafe.Pointer(nv.Pointer())
	src := v.Elem()

	if isScala(elemType.Kind()) {
		shadowCopy(src, p)
		return nv
	}

	switch elemType.Kind() {
	case reflect.Struct:
		copyStruct(src, nv, visited)
	case reflect.Array:
		copyArray(src, nv, visited)
	default:
		nv.Elem().Set(clone(src, visited))
	}

	return nv
}

func cloneSlice(v reflect.Value, visited visitMap) reflect.Value {
	if v.IsNil() {
		return reflect.Zero(v.Type())
	}

	t := v.Type()
	num := v.Len()

	if visited != nil {
		visit := visit{
			p:     v.Pointer(),
			extra: num,
			t:     t,
		}

		if val, ok := visited[visit]; ok {
			return val
		}
	}

	c := v.Cap()
	nv := reflect.MakeSlice(t, num, c)

	if visited != nil {
		visit := visit{
			p:     v.Pointer(),
			extra: num,
			t:     t,
		}
		visited[visit] = nv
	}

	// For scala slice, copy underlying values directly.
	if isScala(t.Elem().Kind()) {
		src := unsafe.Pointer(v.Pointer())
		dst := unsafe.Pointer(nv.Pointer())
		sz := int(t.Elem().Size())
		l := num * sz
		cc := c * sz
		copy((*[math.MaxInt32]byte)(dst)[:l:cc], (*[math.MaxInt32]byte)(src)[:l:cc])
	} else {
		for i := 0; i < num; i++ {
			nv.Index(i).Set(clone(v.Index(i), visited))
		}
	}

	return nv
}

func cloneStruct(v reflect.Value, visited visitMap) reflect.Value {
	t := v.Type()
	nv := reflect.New(t)
	copyStruct(v, nv, visited)
	return nv.Elem()
}

func copyStruct(src, dst reflect.Value, visited visitMap) {
	ptr := unsafe.Pointer(dst.Pointer()) // dst must be a Ptr.
	dst = dst.Elem()
	st := loadStructType(dst.Type())

	// If src is an unexported field value. it's not possible to copy src by value.
	// Copy field value one by one.
	if !dst.CanSet() || !src.CanInterface() {
		t := src.Type()
		num := t.NumField()

		for i := 0; i < num; i++ {
			field := t.Field(i)
			p := unsafe.Pointer(uintptr(ptr) + field.Offset)
			copyStructField(src.Field(i), dst.Field(i), p, visited)
		}

		return
	}

	dst.Set(src)

	// If the struct type is a scala type, a.k.a type without any pointer,
	// there is no need to iterate over fields.
	if len(st.PointerFields) == 0 {
		return
	}

	for _, pf := range st.PointerFields {
		i := int(pf.Index)
		p := unsafe.Pointer(uintptr(ptr) + pf.Offset)
		copyStructField(src.Field(i), dst.Field(i), p, visited)
	}
}

func copyStructField(src, dst reflect.Value, p unsafe.Pointer, visited visitMap) {
	v := clone(src, visited)

	// dst is a public field.
	if dst.CanSet() && v.CanInterface() {
		dst.Set(v)
		return
	}

	shadowCopy(v, p)
}

func shadowCopy(src reflect.Value, p unsafe.Pointer) {
	switch src.Kind() {
	case reflect.Bool:
		*(*bool)(p) = src.Bool()
	case reflect.Int:
		*(*int)(p) = int(src.Int())
	case reflect.Int8:
		*(*int8)(p) = int8(src.Int())
	case reflect.Int16:
		*(*int16)(p) = int16(src.Int())
	case reflect.Int32:
		*(*int32)(p) = int32(src.Int())
	case reflect.Int64:
		*(*int64)(p) = src.Int()
	case reflect.Uint:
		*(*uint)(p) = uint(src.Uint())
	case reflect.Uint8:
		*(*uint8)(p) = uint8(src.Uint())
	case reflect.Uint16:
		*(*uint16)(p) = uint16(src.Uint())
	case reflect.Uint32:
		*(*uint32)(p) = uint32(src.Uint())
	case reflect.Uint64:
		*(*uint64)(p) = src.Uint()
	case reflect.Uintptr:
		*(*uintptr)(p) = uintptr(src.Uint())
	case reflect.Float32:
		*(*float32)(p) = float32(src.Float())
	case reflect.Float64:
		*(*float64)(p) = src.Float()
	case reflect.Complex64:
		*(*complex64)(p) = complex64(src.Complex())
	case reflect.Complex128:
		*(*complex128)(p) = src.Complex()

	case reflect.Array:
		t := src.Type()
		val := reflect.NewAt(t, p).Elem()

		if src.CanInterface() {
			val.Set(src)
			return
		}

		sz := t.Elem().Size()
		num := src.Len()

		for i := 0; i < num; i++ {
			elemPtr := unsafe.Pointer(uintptr(p) + uintptr(i)*sz)
			shadowCopy(src.Index(i), elemPtr)
		}
	case reflect.Chan:
		*((*uintptr)(p)) = src.Pointer()
	case reflect.Func:
		t := src.Type()
		src = copyScalaValue(src)
		val := reflect.NewAt(t, p).Elem()
		val.Set(src)
	case reflect.Interface:
		*((*[2]uintptr)(p)) = src.InterfaceData()
	case reflect.Map:
		*((*uintptr)(p)) = src.Pointer()
	case reflect.Ptr:
		*((*uintptr)(p)) = src.Pointer()
	case reflect.Slice:
		*(*reflect.SliceHeader)(p) = reflect.SliceHeader{
			Data: src.Pointer(),
			Len:  src.Len(),
			Cap:  src.Cap(),
		}
	case reflect.String:
		s := src.String()
		*(*reflect.StringHeader)(p) = *(*reflect.StringHeader)(unsafe.Pointer(&s))
	case reflect.Struct:
		t := src.Type()
		val := reflect.NewAt(t, p).Elem()

		if src.CanInterface() {
			val.Set(src)
			return
		}

		num := t.NumField()

		for i := 0; i < num; i++ {
			field := t.Field(i)
			fieldPtr := unsafe.Pointer(uintptr(p) + field.Offset)
			shadowCopy(src.Field(i), fieldPtr)
		}
	case reflect.UnsafePointer:
		// There is no way to copy unsafe.Pointer value.
		*((*uintptr)(p)) = src.Pointer()

	default:
		panic(fmt.Errorf("go-clone: <bug> impossible type `%v` when cloning private field", src.Type()))
	}
}
