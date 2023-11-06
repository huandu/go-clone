// Copyright 2019 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

package clone

import (
	"fmt"
)

func ExampleSlowly() {
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

	// Output:
	// 1
	// 2
	// 3
	// 1
	// 2
	// 3
	// 1
	// 2
	// 3
	// 1
}

func ExampleClone_tags() {
	type T struct {
		Normal *int
		Foo    *int `clone:"skip"`       // Skip cloning this field so that Foo will be nil in cloned value.
		Bar    *int `clone:"-"`          // "-" is an alias of skip.
		Baz    *int `clone:"shadowcopy"` // Copy this field by value so that Baz will the same pointer as the original one.
	}

	a := 1
	t := &T{
		Normal: &a,
		Foo:    &a,
		Bar:    &a,
		Baz:    &a,
	}
	v := Clone(t).(*T)

	fmt.Println(v.Normal == t.Normal) // false
	fmt.Println(v.Foo == nil)         // true
	fmt.Println(v.Bar == nil)         // true
	fmt.Println(v.Baz == t.Baz)       // true

	// Output:
	// false
	// true
	// true
	// true
}
