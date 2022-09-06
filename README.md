# go-clone: Deep clone any Go data

[![Go](https://github.com/huandu/go-clone/workflows/Go/badge.svg)](https://github.com/huandu/go-clone/actions)
[![Go Doc](https://godoc.org/github.com/huandu/go-clone?status.svg)](https://pkg.go.dev/github.com/huandu/go-clone)
[![Go Report](https://goreportcard.com/badge/github.com/huandu/go-clone)](https://goreportcard.com/report/github.com/huandu/go-clone)
[![Coverage Status](https://coveralls.io/repos/github/huandu/go-clone/badge.svg?branch=master)](https://coveralls.io/github/huandu/go-clone?branch=master)

Package `clone` provides functions to deep clone any Go data.
It also provides a wrapper to protect a pointer from any unexpected mutation.

`Clone`/`Slowly` can clone unexported fields and "no-copy" structs as well. Use this feature wisely.

## Install

Use `go get` to install this package.

```shell
go get github.com/huandu/go-clone
```

## Usage

### `Clone` and `Slowly`

If we want to clone any Go value, use `Clone`.

```go
t := &T{...}
v := clone.Clone(t).(*T)
reflect.DeepEqual(t, v) // true
```

For the sake of performance, `Clone` doesn't deal with values containing pointer cycles.
If we need to clone such values, use `Slowly` instead.

```go
type ListNode struct {
    Data int
    Next *ListNode
}
node1 := &ListNode{
    Data: 1,
}
node2 := &ListNode{
    Data: 2,
}
node3 := &ListNode{
    Data: 3,
}
node1.Next = node2
node2.Next = node3
node3.Next = node1

// We must use `Slowly` to clone a circular linked list.
node := Slowly(node1).(*ListNode)

for i := 0; i < 10; i++ {
    fmt.Println(node.Data)
    node = node.Next
}
```

### Mark struct type as scalar

Some struct types can be considered as scalar.

A well-known case is `time.Time`.
Although there is a pointer `loc *time.Location` inside `time.Time`, we always use `time.Time` by value in all methods.
When cloning `time.Time`, it should be OK to return a shadow copy.

Currently, following types are marked as scalar by default.

- `time.Time`
- `reflect.Value`

If there is any type defined in built-in package should be considered as scalar, please open new issue to let me know.
I will update the default.

If there is any custom type should be considered as scalar, call `MarkAsScalar` to mark it manually. See [MarkAsScalar sample code](https://pkg.go.dev/github.com/huandu/go-clone#example-MarkAsScalar) for more details.

### Mark pointer type as opaque

Some pointer values are used as enumerable const values.

A well-known case is `elliptic.Curve`. In package `crypto/tls`, curve type of a certificate is checked by comparing values to pre-defined curve values, e.g. `elliptic.P521()`. In this case, the curve values, which are pointers or structs, cannot be cloned deeply.

Currently, following types are marked as scalar by default.

- `elliptic.Curve`, which is `*elliptic.CurveParam` or `elliptic.p256Curve`.
- `reflect.Type`, which is `*reflect.rtype` defined in `runtime`.

If there is any pointer type defined in built-in package should be considered as opaque, please open new issue to let me know.
I will update the default.

If there is any custom pointer type should be considered as opaque, call `MarkAsOpaquePointer` to mark it manually. See [MarkAsOpaquePointer sample code](https://pkg.go.dev/github.com/huandu/go-clone#example-MarkAsOpaquePointer) for more details.

### Clone "no-copy" types defined in `sync` and `sync/atomic`

There are some "no-copy" types like `sync.Mutex`, `atomic.Value`, etc.
They cannot be cloned by copying all fields one by one, but we can alloc a new zero value and call methods to do proper initialization.

Currently, all "no-copy" types defined in `sync` and `sync/atomic` can be cloned properly using following strategies.

- `sync.Mutex`: Cloned value is a newly allocated zero mutex.
- `sync.RWMutex`: Cloned value is a newly allocated zero mutex.
- `sync.WaitGroup`: Cloned value is a newly allocated zero wait group.
- `sync.Cond`: Cloned value is a cond with a newly allocated zero lock.
- `sync.Pool`: Cloned value is an empty pool with the same `New` function.
- `sync.Map`: Cloned value is a sync map with cloned key/value pairs.
- `sync.Once`: Cloned value is a once type with the same done flag.
- `atomic.Value`/`atomic.Bool`/`atomic.Int32`/`atomic.Int64`/`atomic.Uint32`/`atomic.Uint64`/`atomic.Uintptr`: Cloned value is a new atomic value with the same value.

If there is any type defined in built-in package should be considered as "no-copy" types, please open new issue to let me know.
I will update the default.

### Set custom clone functions

If default clone strategy doesn't work for a struct type, we can call `SetCustomFunc` to register a custom clone function.
`Clone` and `Slowly` can be used in custom clone functions.

See [SetCustomFunc sample code](https://pkg.go.dev/github.com/huandu/go-clone#example-SetCustomFunc) for more details.

### Clone `atomic.Pointer[T]`

As there is no way to predefine a custom clone function for generic type `atomic.Pointer[T]`, cloning such atomic type is not supported by default. If we want to support it, we need to register a custom clone function manually.

Suppose we instantiate `atomic.Pointer[T]` with type `MyType1` and `MyType2` in a project, and then we can register custom clone functions like following.

```go
// registerAtomicPointer registers a custom clone function for atomic.Pointer[T].
func registerAtomicPointer[T any]() {
    clone.SetCustomFunc(reflect.TypeOf(atomic.Pointer[T]{}), func(old, new reflect.Value) {
        if !old.CanAddr() {
            return
        }

        // Clone value inside atomic.Pointer[T].
        oldValue := old.Addr().Interface().(*atomic.Pointer[T])
        newValue := new.Addr().Interface().(*atomic.Pointer[T])
        v := oldValue.Load()
        newValue.Store(v)
    })
}

func init() {
    // Register all instantiated atomic.Pointer[T] types in this project.
    registerAtomicPointer[MyType1]()
    registerAtomicPointer[MyType2]()
}
```

### `Wrap`, `Unwrap` and `Undo`

Package `clone` provides `Wrap`/`Unwrap` functions to protect a pointer value from any unexpected mutation.
It's useful when we want to protect a variable which should be immutable by design,
e.g. global config, the value stored in context, the value sent to a chan, etc.

```go
// Suppose we have a type T defined as following.
//     type T struct {
//         Foo int
//     }
v := &T{
    Foo: 123,
}
w := Wrap(v).(*T) // Wrap value to protect it.

// Use w freely. The type of w is the same as that of v.

// It's OK to modify w. The change will not affect v.
w.Foo = 456
fmt.Println(w.Foo) // 456
fmt.Println(v.Foo) // 123

// Once we need the original value stored in w, call `Unwrap`.
orig := Unwrap(w).(*T)
fmt.Println(orig == v) // true
fmt.Println(orig.Foo)  // 123

// Or, we can simply undo any change made in w.
// Note that `Undo` is significantly slower than `Unwrap`, thus
// the latter is always preferred.
Undo(w)
fmt.Println(w.Foo) // 123
```

## Performance

Here is the performance data running on my MacBook Pro.

```text
MacBook Pro (15-inch, 2019)
Processor: 2.6 GHz Intel Core i7

go 1.19
goos: darwin
goarch: amd64
pkg: github.com/huandu/go-clone
cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
BenchmarkSimpleClone-12          7903873               142.9 ns/op            24 B/op          1 allocs/op
BenchmarkComplexClone-12          590836              1755 ns/op            1488 B/op         21 allocs/op
BenchmarkUnwrap-12              14988664                71.46 ns/op            0 B/op          0 allocs/op
BenchmarkSimpleWrap-12           3823450               304.4 ns/op            72 B/op          2 allocs/op
BenchmarkComplexWrap-12           867642              1197 ns/op             736 B/op         15 allocs/op
```

## License

This package is licensed under MIT license. See LICENSE for details.
