// Licensed under the MIT license that can be found in the LICENSE file.

package clone

import (
	"reflect"
	"testing"
)

type namedChannel chan int
type namedReceiveChannel <-chan int
type namedSendChannel chan<- int

type channelAPI[T any] struct {
	name  string
	clone func(T) T
}

func TestGenericChannelDirections(t *testing.T) {
	testGenericChannelType[chan int](t, "buffered")
	testGenericChannelType[<-chan int](t, "buffered")
	testGenericChannelType[namedSendChannel](t, "buffered")
	testGenericChannelType[namedChannel](t, "nil")
}

func testGenericChannelType[T any](t *testing.T, states ...string) {
	testChannelType(t, []channelAPI[T]{{"Clone", Clone[T]}, {"Slowly", Slowly[T]}}, states...)
}

func testChannelType[T any](t *testing.T, apis []channelAPI[T], states ...string) {
	typ := reflect.TypeOf((*T)(nil)).Elem()
	t.Run(typ.String(), func(t *testing.T) {
		for _, api := range apis {
			t.Run(api.name, func(t *testing.T) {
				for _, state := range states {
					t.Run(state, func(t *testing.T) {
						defer failChannelPanic(t)
						source := channelSource(state)
						input := reflect.ValueOf(source).Convert(typ).Interface().(T)
						output := reflect.ValueOf(api.clone(input))
						checkChannelClone(t, source, output, typ)
					})
				}
			})
		}
	})
}

func failChannelPanic(t *testing.T) {
	t.Helper()
	if failure := recover(); failure != nil {
		t.Fatalf("cloning a channel panicked: %v", failure)
	}
}

func channelSource(state string) chan int {
	if state == "nil" {
		return nil
	}
	buffer := 0
	if state == "buffered" {
		buffer = 2
	}
	ch := make(chan int, buffer)
	if buffer > 0 {
		ch <- 41
	}
	return ch
}

func checkChannelClone(t *testing.T, source chan int, output reflect.Value, typ reflect.Type) {
	t.Helper()
	if output.Type() != typ || output.IsNil() || output.Cap() != cap(source) || output.Len() != 0 {
		t.Fatalf("clone type=%v nil=%v len=%d cap=%d; want %v, non-nil, 0, %d", output.Type(), output.IsNil(), output.Len(), output.Cap(), typ, cap(source))
	}
	if output.Pointer() == reflect.ValueOf(source).Pointer() {
		t.Fatal("clone reused the source channel")
	}
	if cap(source) > 0 {
		select {
		case value := <-source:
			if value != 41 {
				t.Fatalf("source payload = %d; want 41", value)
			}
		default:
			t.Fatal("clone consumed the source payload")
		}
	}
	if typ.ChanDir()&reflect.RecvDir != 0 {
		if value, _ := output.TryRecv(); value.IsValid() {
			t.Fatal("new channel is readable; want empty and open")
		}
	}
	if cap(source) > 0 && typ.ChanDir()&reflect.SendDir != 0 {
		if !output.TrySend(reflect.ValueOf(99)) {
			t.Fatal("cannot send to the empty clone")
		}
	}
	select {
	case <-source:
		t.Fatal("source is readable after draining; clone must remain independent")
	default:
	}
}
