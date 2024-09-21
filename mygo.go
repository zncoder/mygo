package mygo

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/zncoder/check"
)

func FileExist(filename string) bool {
	_, err := os.Stat(filename)
	if err == nil {
		return true
	}
	check.T(errors.Is(err, fs.ErrNotExist)).F("stat", "filename", filename)
	return false
}

func FileSize(filename string) (bool, int64) {
	fi, err := os.Stat(filename)
	if err == nil {
		return true, fi.Size()
	}
	check.T(errors.Is(err, fs.ErrNotExist)).F("stat", "filename", filename)
	return false, 0
}

func FileMode(filename string) fs.FileMode {
	if st, err := os.Lstat(filename); err == nil {
		return st.Mode()
	}
	return 0
}

func IsSymlink(filename string) bool {
	return FileMode(filename)&os.ModeSymlink != 0
}

func IsDir(filename string) bool {
	return FileMode(filename)&os.ModeDir != 0
}

func ErrNotExist(err error) bool {
	return errors.Is(err, fs.ErrNotExist)
}

func GuessUTF8File(filename string) bool {
	f, err := os.Open(filename)
	if err != nil {
		return false
	}
	defer f.Close()
	var buf [400]byte
	n, err := f.Read(buf[:])
	n -= utf8.UTFMax
	if err != nil {
		return false
	}
	for i := 0; i < n; {
		r, size := utf8.DecodeRune(buf[i:])
		if r == utf8.RuneError {
			return false
		}
		i += size
	}
	return true
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
	C         *exec.Cmd
	trace     bool
	ignoreErr bool
}

func NewCmd(name string, args ...string) Cmd {
	c := exec.Command(name, args...)
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	return Cmd{C: c}
}

func (c Cmd) Silent(silent bool) Cmd {
	if silent {
		c.C.Stderr = nil
		c.C.Stdout = nil
	}
	return c
}

func (c Cmd) IgnoreErr(ignoreErr bool) Cmd {
	c.ignoreErr = ignoreErr
	return c
}

func (c Cmd) Trace() Cmd {
	c.trace = true
	return c
}

func (c Cmd) Run() {
	c.showTrace()
	check.E(c.C.Run()).S(c.ignoreErr).F("cmd run failed", "args", c.C.Args)
}

func (c Cmd) RunWithExitCode() int {
	c.showTrace()
	c.C.Run()
	return c.C.ProcessState.ExitCode()
}

func (c Cmd) Start() *os.Process {
	c.showTrace()
	check.E(c.C.Start()).S(c.ignoreErr).F("cmd start failed", "args", c.C.Args)
	return c.C.Process
}

func (c Cmd) showTrace() {
	if c.trace {
		slog.Info("run cmd", "args", c.C.Args)
	}
}

func (c Cmd) Stdout() []byte {
	c.showTrace()
	c.C.Stdout = nil
	return check.V(c.C.Output()).S(c.ignoreErr).F("cmd stdout failed", "args", c.C.Args)
}

func (c Cmd) Interactive() {
	c.showTrace()
	check.T(c.C.Stderr != nil && c.C.Stdout != nil).F("cannot be silent")
	c.C.Stdin = os.Stdin
	check.E(c.C.Run()).S(c.ignoreErr).F("cmd interactive run failed", "args", c.C.Args)
}

var ignoredExts = []string{".o", ".so", ".exe", ".dylib", ".test", ".out"}

func IgnoreFile(filename string) bool {
	if strings.Contains(filename, "/.") || strings.HasSuffix(filename, "~") {
		return true
	}
	ext := strings.ToLower(filepath.Ext(filename))
	if slices.Contains(ignoredExts, ext) {
		return true
	}
	if mode := FileMode(filename); mode&(os.ModeDir|os.ModeSymlink) != 0 {
		return true
	}
	return false
}

func HomeFile(filename string) string {
	check.T(strings.HasPrefix(filename, "~/")).F("filename not start with ~/", "filename", filename)
	home := check.V(os.UserHomeDir()).F("get home dir")
	return filepath.Join(home, filename[2:])
}

func Type(arg any) string {
	return fmt.Sprintf("%T", arg)
}
