package clone

import "fmt"

func ExampleWrap() {
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

	// Output:
	// 456
	// 123
	// true
	// 123
	// 123
}
