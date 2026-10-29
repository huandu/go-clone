// Licensed under the MIT license that can be found in the LICENSE file.

package clone

import (
	"reflect"
	"testing"
	"unsafe"
)

type namedChannel chan int
type namedReceiveChannel <-chan int
type namedSendChannel chan<- int

func TestChannelDirections(t *testing.T) {
	for _, api := range []struct {
		name  string
		clone func(interface{}) interface{}
	}{{"Clone", Clone}, {"Slowly", Slowly}} {
		t.Run(api.name, func(t *testing.T) {
			for _, tc := range []struct {
				typ   reflect.Type
				state string
			}{
				{reflect.TypeOf((chan int)(nil)), "buffered"},
				{reflect.TypeOf((<-chan int)(nil)), "buffered"},
				{reflect.TypeOf((chan<- int)(nil)), "buffered"},
				{reflect.TypeOf(namedChannel(nil)), "buffered"},
				{reflect.TypeOf(namedReceiveChannel(nil)), "buffered"},
				{reflect.TypeOf(namedSendChannel(nil)), "buffered"},
				{reflect.TypeOf((<-chan int)(nil)), "unbuffered"},
				{reflect.TypeOf((chan<- int)(nil)), "unbuffered"},
				{reflect.TypeOf((chan int)(nil)), "nil"},
				{reflect.TypeOf(namedChannel(nil)), "nil"},
			} {
				t.Run(tc.typ.String()+"/"+tc.state, func(t *testing.T) {
					defer failChannelPanic(t)
					source := channelSource(tc.state)
					input := reflect.ValueOf(source).Convert(tc.typ)
					output := reflect.ValueOf(api.clone(input.Interface()))
					checkChannelClone(t, source, output, tc.typ, nil)
				})
			}
		})
	}
}

func TestCustomChannelAllocator(t *testing.T) {
	for _, slow := range []bool{false, true} {
		name := "Clone"
		typ := reflect.TypeOf(namedReceiveChannel(nil))
		if slow {
			name = "CloneSlowly"
			typ = reflect.TypeOf(namedSendChannel(nil))
		}
		t.Run(name, func(t *testing.T) {
			for _, route := range []string{"own", "inherited same pool", "inherited different pool"} {
				t.Run(route, func(t *testing.T) {
					defer failChannelPanic(t)
					var pools [2]int
					pool := unsafe.Pointer(&pools[0])
					var allocated chan int
					calls := 0
					parent := NewAllocator(pool, &AllocatorMethods{
						MakeChan: func(gotPool unsafe.Pointer, gotType reflect.Type, buffer int) reflect.Value {
							calls++
							if gotPool != pool || gotType != typ || buffer != 2 {
								t.Fatalf("MakeChan received pool=%p type=%v buffer=%d; want %p, %v, 2", gotPool, gotType, buffer, pool, typ)
							}
							allocated = make(chan int, buffer)
							return reflect.ValueOf(allocated).Convert(gotType)
						},
					})
					allocator := parent
					if route == "inherited same pool" {
						allocator = NewAllocator(pool, &AllocatorMethods{Parent: parent})
					} else if route == "inherited different pool" {
						allocator = NewAllocator(unsafe.Pointer(&pools[1]), &AllocatorMethods{Parent: parent})
					}
					source := channelSource("buffered")
					input := reflect.ValueOf(source).Convert(typ)
					var output reflect.Value
					if slow {
						output = allocator.CloneSlowly(input)
					} else {
						output = allocator.Clone(input)
					}
					if calls != 1 {
						t.Fatalf("MakeChan called %d times; want 1", calls)
					}
					checkChannelClone(t, source, output, typ, allocated)
				})
			}
		})
	}
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

func checkChannelClone(t *testing.T, source chan int, output reflect.Value, typ reflect.Type, allocated chan int) {
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
		if allocated != nil {
			select {
			case value := <-allocated:
				if value != 99 {
					t.Fatalf("clone payload = %d; want 99", value)
				}
			default:
				t.Fatal("custom allocator did not expose the clone's channel")
			}
		}
	}
	select {
	case <-source:
		t.Fatal("source is readable after draining; clone must remain independent")
	default:
	}
}
