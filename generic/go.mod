module github.com/huandu/go-clone/generic

go 1.18

require (
	github.com/huandu/go-assert v1.1.5
	github.com/huandu/go-clone v1.7.3
)

require github.com/davecgh/go-spew v1.1.1 // indirect

// Test the generic API against the working-tree copy of the root module instead
// of the published release. Without this, `go test ./...` inside generic/ builds
// github.com/huandu/go-clone from the module cache, so the submodule tests cannot
// observe any change made to the root module in this repository.
//
// A replace directive applies only to the main module that declares it. It is not
// inherited by downstream modules and does not affect what `go get` resolves, so
// this is a local development and CI aid only. Consumers of
// github.com/huandu/go-clone/generic still build against the required version.
replace github.com/huandu/go-clone => ../
