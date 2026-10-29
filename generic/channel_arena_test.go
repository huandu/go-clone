// Licensed under the MIT license that can be found in the LICENSE file.

//go:build go1.20 && goexperiment.arenas
// +build go1.20,goexperiment.arenas

package clone

import (
	"arena"
	"testing"
)

func TestArenaChannelDirections(t *testing.T) {
	a := arena.NewArena()
	defer a.Free()
	testArenaChannelType[chan int](t, a, "buffered", "nil")
	testArenaChannelType[<-chan int](t, a, "buffered", "unbuffered")
	testArenaChannelType[chan<- int](t, a, "buffered", "unbuffered")
	testArenaChannelType[namedChannel](t, a, "buffered", "nil")
	testArenaChannelType[namedReceiveChannel](t, a, "buffered")
	testArenaChannelType[namedSendChannel](t, a, "buffered")
}

func testArenaChannelType[T any](t *testing.T, a *arena.Arena, states ...string) {
	testChannelType(t, []channelAPI[T]{
		{"ArenaClone", func(v T) T { return ArenaClone(a, v) }},
		{"ArenaCloneSlowly", func(v T) T { return ArenaCloneSlowly(a, v) }},
	}, states...)
}
