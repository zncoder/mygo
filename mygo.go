package mygo

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"reflect"
	"regexp"
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

type OP struct {
	Alias string
	Name  string
	Fn    func()
}

type OPMap map[string]*OP

func (om OPMap) Run(alias string) { om[alias].Fn() }

func (om OPMap) Add(alias string, fn func()) {
	check.T(om[alias] == nil).P("alias in use", "alias", alias)
	om[alias] = &OP{Alias: alias, Name: alias, Fn: fn}
}

// BuildOPMap extracts exported methods of opRecv to a map,
// so that the methods can be called by the name or alias.
// An example opRecv is,
//
//	type GitOP struct {}
//	func (op GitOP) CM_Commit() {...}
//	func (op GitOP) Status() {...}
//
// BuildOPMap[GitOP]() returns an OPMap,
//
//	{
//	    "cm": OP{Alias: "cm", Name: "commit", Fn: wrapper_of_GitOP.CM_Commit,
//	    "status": OP{Alias: "status", Name: "status", Fn: wrapper_of_GitOP.Status,
//	}
//
// then we can call the op by alias,
//
//	var gitop OPMap = BuildOPMap[GitOP]()
//	gitop.Run("cm")
//
// we can add additional methods manually to OPMap,
//
//	gitop.Add("log", gitLog)
func BuildOPMap[T any]() OPMap {
	ops := make(OPMap)
	nameRe := regexp.MustCompile(`^([A-Z]+_)?([A-Z].*)$`)
	var op T
	rt := reflect.TypeOf(op)
	for i := 0; i < rt.NumMethod(); i++ {
		alias, name, fn := buildMethod[T](rt.Method(i), nameRe)
		_, ok := ops[alias]
		check.T(!ok).F("alias in use", "alias", alias)
		ops[alias] = &OP{Alias: alias, Name: name, Fn: func() { fn(op) }}
	}
	return ops
}

func buildMethod[T any](m reflect.Method, nameRe *regexp.Regexp) (alias, name string, fn func(T)) {
	mo := nameRe.FindStringSubmatch(m.Name)
	check.T(mo != nil).F("invalid op method", "name", m.Name)
	if mo[1] != "" {
		alias = strings.ToLower(mo[1][:len(mo[1])-1])
	} else {
		alias = strings.ToLower(mo[2])
	}
	name = mo[2]
	check.T(name != "").F("empty method name", "name", m.Name)
	fn = m.Func.Interface().(func(T))
	return alias, name, fn
}

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
