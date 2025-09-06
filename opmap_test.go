package mygo

import (
	"testing"
)

type Foo struct{}

var hello, copy int

func (Foo) Hello()   { hello++ }
func (Foo) CP_Copy() { copy++ }

func TestBuildOPMap(t *testing.T) {
	ops := BuildOPMap[Foo]()
	ops.mustRun("hello")
	if hello != 1 {
		t.Fatal("hello not called")
	}
	ops.mustRun("cp")
	if copy != 1 {
		t.Fatal("cp not called")
	}
	ops.fuzzyRun("ll")
	if hello != 2 {
		t.Fatal("hello not called")
	}
}
