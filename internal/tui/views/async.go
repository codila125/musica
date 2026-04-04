package views

import "sync/atomic"

var requestIDCounter int64

func nextRequestID() int64 {
	return atomic.AddInt64(&requestIDCounter, 1)
}
