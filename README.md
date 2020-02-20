# go-clone: Deep clone any Go data #

[![Build Status](https://travis-ci.org/huandu/go-clone.svg?branch=master)](https://travis-ci.org/huandu/go-clone)
[![GoDoc](https://godoc.org/github.com/huandu/go-clone?status.svg)](https://godoc.org/github.com/huandu/go-clone)
[![Go Report](https://goreportcard.com/badge/github.com/huandu/go-clone)](https://goreportcard.com/report/github.com/huandu/go-clone)
[![Coverage Status](https://coveralls.io/repos/github/huandu/go-clone/badge.svg?branch=master)](https://coveralls.io/github/huandu/go-clone?branch=master)


Package `clone` provides functions to deep clone any Go data.
It also provides a wrapper to protect a pointer from any unexpected mutation.

`Clone`/`Slowly` can clone unexported fields of any struct. Use this feature wisely.

## Install ##

Use `go get` to install this package.

    go get github.com/huandu/go-clone

## Usage ##

### `Clone` and `Slowly` ###

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

### Mark struct type as scalar ###

Some struct types can be considered as scalar.

A well-known case is `time.Time`.
Although there is a pointer `loc *time.Location` inside `time.Time`, we always use `time.Time` by value in all methods.
When cloning `time.Time`, it should be OK to return a shadow copy.

Currently, following types are marked as scalar by default.

* `time.Time`
* `reflect.Value`

If there is any type defined in built-in package should be considered as scalar, please open new issue to let me know.
I will update the default.

If there is any custom type should be considered as scalar, call `MarkAsScalar` to mark it manually. See [MarkAsScalar sample code](https://godoc.org/github.com/huandu/go-clone#example-MarkAsScalar) for more details.

### `Wrap`, `Unwrap` and `Undo` ###

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

## Performance ##

Here is the performance data running on my MacBook Pro.

```
MacBook Pro (15-inch, 2019)
Processor: 2.6 GHz Intel Core i7

go 1.13.7
goos: darwin
goarch: amd64
pkg: github.com/huandu/go-clone

BenchmarkSimpleClone-12     	10254127	       108 ns/op	      32 B/op	       1 allocs/op
BenchmarkComplexClone-12    	  667335	      1831 ns/op	    1472 B/op	      22 allocs/op
BenchmarkUnwrap-12          	13315618	        91.2 ns/op	       0 B/op	       0 allocs/op
BenchmarkSimpleWrap-12      	 5119616	       238 ns/op	      48 B/op	       1 allocs/op
BenchmarkComplexWrap-12     	 1000000	      1158 ns/op	     688 B/op	      13 allocs/op
```

## Similar packages ##

* Package [encoding/gob](https://golang.org/pkg/encoding/gob/): Gob encoder/decoder can be used to clone Go data. However, it's extremely slow.
* Package [github.com/jinzhu/copier](https://github.com/jinzhu/copier): Copy data by field name. It doesn't work with values containing pointer cycles.
* Package [github.com/ulule/deepcopier](https://github.com/ulule/deepcopier): Another copier.

## License ##

This package is licensed under MIT license. See LICENSE for details.
