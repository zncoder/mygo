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
	"time"
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

func IsFileFresh(filename string, expire time.Time) bool {
	st, ok := check.V(os.Stat(filename)).S().K("stat")
	if !ok {
		return false
	}
	return st.ModTime().After(expire)
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
	C                *exec.Cmd
	trace, ignoreErr bool
}

func NewCmd(name string, args ...string) Cmd {
	c := exec.Command(name, args...)
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	return Cmd{C: c}
}

func (c Cmd) S() Cmd {
	c.C.Stderr = nil
	c.C.Stdout = nil
	return c
}

func (c Cmd) T() Cmd {
	c.trace = true
	return c
}

func (c Cmd) I() Cmd {
	c.ignoreErr = true
	return c
}

func (c Cmd) Run() {
	c.showTrace()
	err := c.C.Run()
	if !c.ignoreErr {
		check.E(err).F("cmd run failed", "args", c.C.Args)
	}
}

func (c Cmd) Code() int {
	c.showTrace()
	c.C.Run()
	return c.C.ProcessState.ExitCode()
}

func (c Cmd) Start() *os.Process {
	c.showTrace()
	err := c.C.Start()
	if !c.ignoreErr {
		check.E(err).F("cmd start failed", "args", c.C.Args)
	}
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
	b, err := c.C.Output()
	if !c.ignoreErr {
		check.E(err).F("cmd stdout failed", "args", c.C.Args)
	}
	return b
}

func (c Cmd) Interactive() {
	c.showTrace()
	check.T(c.C.Stderr != nil && c.C.Stdout != nil && !c.ignoreErr).F("cannot be silent")
	c.C.Stdin = os.Stdin
	check.E(c.C.Run()).F("cmd interactive run failed", "args", c.C.Args)
}

var ignoredExts = []string{".o", ".so", ".exe", ".dylib", ".test", ".out"}

func IgnoreFilename(filename string, exts ...string) bool {
	if strings.HasPrefix(filename, ".") || strings.Contains(filename, "/.") || strings.HasSuffix(filename, "~") {
		return true
	}
	ext := strings.ToLower(filepath.Ext(filename))
	if slices.Contains(ignoredExts, ext) || slices.Contains(exts, ext) {
		return true
	}
	return false
}

func IgnoreRegularFile(filename string, exts ...string) bool {
	if IgnoreFilename(filename, exts...) {
		return true
	}
	if mode := FileMode(filename); !mode.IsRegular() {
		return true
	}
	return false
}

func HomeFile(filename string) string {
	if !strings.HasPrefix(filename, "~/") {
		return filename
	}
	home := check.V(os.UserHomeDir()).F("get home dir")
	return filepath.Join(home, filename[2:])
}

func Type(arg any) string {
	return fmt.Sprintf("%T", arg)
}
