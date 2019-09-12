// Copyright 2019 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

// Package clone provides functions to deep clone any Go data.
// It also provides a wrapper to protect a pointer from any unexpected mutation.
package clone

import (
	"reflect"
)

// Clone recursively deep clone v to a new value.
// It assumes that there is no recursive pointers in v,
// e.g. v has a pointer points to v itself.
func Clone(v interface{}) interface{} {
	if v == nil {
		return v
	}

	val := reflect.ValueOf(v)
	cloned := clone(val, nil)
	return cloned.Interface()
}

// Slowly recursively deep clone v to a new value.
// It marks all cloned values internally, thus it can clone
// recursive pointers in v.
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
	switch v.Kind() {
	case reflect.Array:
		return cloneArray(v, visited)
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
		return v
	}
}

func cloneArray(v reflect.Value, visited visitMap) reflect.Value {
	t := v.Type()
	num := v.Len()
	nv := reflect.New(t).Elem()

	for i := 0; i < num; i++ {
		nv.Index(i).Set(clone(v.Index(i), visited))
	}

	return nv
}

func cloneInterface(v reflect.Value, visited visitMap) reflect.Value {
	if v.IsNil() {
		return v
	}

	t := v.Type()
	return clone(v.Elem(), visited).Convert(t)
}

func cloneMap(v reflect.Value, visited visitMap) reflect.Value {
	if v.IsNil() {
		return v
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
		return v
	}

	if t := v.Type(); visited != nil {
		visit := visit{
			p: v.Pointer(),
			t: t,
		}

		if val, ok := visited[visit]; ok {
			return val
		}

		nv := reflect.New(t.Elem())
		visited[visit] = nv
		nv.Elem().Set(clone(v.Elem(), visited))
		return nv
	}

	return clone(v.Elem(), visited).Addr()
}

func cloneSlice(v reflect.Value, visited visitMap) reflect.Value {
	if v.IsNil() {
		return v
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

	nv := reflect.MakeSlice(t, num, v.Cap())

	if visited != nil {
		visit := visit{
			p:     v.Pointer(),
			extra: num,
			t:     t,
		}
		visited[visit] = nv
	}

	for i := 0; i < num; i++ {
		nv.Index(i).Set(clone(v.Index(i), visited))
	}

	return nv
}

func cloneStruct(v reflect.Value, visited visitMap) reflect.Value {
	t := v.Type()
	nv := reflect.New(t).Elem()
	copyStruct(v, nv, visited)
	return nv
}

func copyStruct(src, dst reflect.Value, visited visitMap) {
	num := src.NumField()

	for i := 0; i < num; i++ {
		field := dst.Field(i)

		if !field.CanSet() {
			continue
		}

		field.Set(clone(src.Field(i), visited))
	}
}
