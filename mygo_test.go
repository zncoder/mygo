package mygo_test

import (
	"testing"

	"github.com/zncoder/mygo"
)

type Foo struct{}

var hello, copy int

func (Foo) Hello()   { hello++ }
func (Foo) CP_Copy() { copy++ }

func TestBuildOPMap(t *testing.T) {
	ops := mygo.BuildOPMap[Foo]()
	ops.Run("hello")
	if hello != 1 {
		t.Fatal("hello not called")
	}
	ops.Run("cp")
	if copy != 1 {
		t.Fatal("cp not called")
	}
}
