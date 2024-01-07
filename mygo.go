package mygo

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/zncoder/check"
)

// ParseFlag specifies non-flag args.
// Ex. ParseFlag("required_arg0", "[optional_arg1]")
// Optional args must appear after required args.
func ParseFlag(args ...string) {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s %s\n", os.Args[0], strings.Join(args, " "))
		flag.PrintDefaults()
	}
	flag.Parse()

	var opt bool
	n := 0
	for _, arg := range args {
		if !strings.HasPrefix(arg, "[") {
			n++
			check.T(!opt).F("required arg appears after optional args", "args", args)
		} else {
			opt = true
		}
	}
	check.T(n <= flag.NArg()).F("not enough required args", "args", args, "flag_args", flag.Args())
}

func FileExist(filename string) bool {
	_, err := os.Stat(filename)
	if err == nil {
		return true
	}
	check.T(errors.Is(err, fs.ErrNotExist)).P("stat", "filename", filename)
	return false
}

type Cmd struct {
	c     *exec.Cmd
	trace bool
}

func NewCmd(name string, args ...string) Cmd {
	c := exec.Command(name, args...)
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	return Cmd{c: c}
}

func (c Cmd) Silent() Cmd {
	c.c.Stderr = nil
	c.c.Stdout = nil
	return c
}

func (c Cmd) Trace() Cmd {
	c.trace = true
	return c
}

func (c Cmd) Run() error {
	c.showTrace()
	return c.c.Run()
}

func (c Cmd) showTrace() {
	if c.trace {
		slog.Info("run cmd", "args", c.c.Args)
	}
}

func (c Cmd) Stdout() ([]byte, error) {
	c.showTrace()
	c.c.Stdout = nil
	return c.c.Output()
}

func (c Cmd) Interactive() error {
	c.showTrace()
	check.T(c.c.Stderr != nil && c.c.Stdout != nil).F("cannot be silent")
	c.c.Stdin = os.Stdin
	return c.c.Run()
}
