package api

import (
	"errors"
	"testing"
)

func TestWrapAndKindOf(t *testing.T) {
	err := Wrap(ErrorKindNetwork, "op", errors.New("boom"))
	if KindOf(err) != ErrorKindNetwork {
		t.Fatalf("expected network error kind")
	}
}

func TestKindOfUnknown(t *testing.T) {
	if KindOf(errors.New("x")) != ErrorKindUnknown {
		t.Fatalf("expected unknown kind")
	}
}
