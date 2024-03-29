package mygo_test

import (
	"testing"

	"github.com/zncoder/mygo"
)

func TestGuessUTF8File(t *testing.T) {
	if !mygo.GuessUTF8File("/etc/passwd") {
		t.Fatal("is text file")
	}
	if mygo.GuessUTF8File("/bin/ls") {
		t.Fatal("is binary file")
	}
}
