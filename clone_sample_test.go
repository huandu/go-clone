package clone

import "fmt"

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
