package mygo

import (
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"

	"github.com/zncoder/check"
)

func FileExist(filename string) bool {
	_, err := os.Stat(filename)
	if err == nil {
		return true
	}
	check.T(errors.Is(err, fs.ErrNotExist)).P("stat", "filename", filename)
	return false
}

func FileSize(filename string) (bool, int64) {
	fi, err := os.Stat(filename)
	if err == nil {
		return true, fi.Size()
	}
	check.T(errors.Is(err, fs.ErrNotExist)).P("stat", "filename", filename)
	return false, 0
}

func IsSymlink(filename string) bool {
	if st, err := os.Lstat(filename); err == nil {
		return st.Mode()&os.ModeSymlink != 0
	}
	return false
}

func ReadLastLink(name string) string {
	origName := name
	for i := 0; i < 20; i++ {
		p, err := os.Readlink(name)
		if err != nil {
			return name
		}
		name = p
	}
	check.T(false).F("readlastlink too many symlinks", "origname", origName, "name", name)
	return origName
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

func (c Cmd) Silent(silent bool) Cmd {
	if silent {
		c.c.Stderr = nil
		c.c.Stdout = nil
	} else if c.c.Stderr == nil {
		c.c.Stderr = os.Stderr
		c.c.Stdout = os.Stdout
	}
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
