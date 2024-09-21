package mygo_test

import (
	"os"
	"testing"

	"github.com/zncoder/mygo"
)

type Handler struct {
}

type Arg struct {
	Name string
}

type Result struct {
	Len int
}

func (h Handler) Handle(arg *Arg) *Result {
	return &Result{Len: len(arg.Name)}
}

func TestLoop(t *testing.T) {
	f, err := os.CreateTemp("", "testloop")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	rs := mygo.NewUnixRPCServer[Arg, Result](f.Name(), Handler{})
	go rs.Loop()

	ur := mygo.NewUnixRPC[Arg, Result](f.Name())
	defer ur.Close()
	result := ur.Call(&Arg{Name: "hello"})
	if result.Len != 5 {
		t.Fatalf("expect 5, got %d", result.Len)
	}
}
