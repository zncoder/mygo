package mygo

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
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

func (om OPMap) Run(alias string) {
	op, ok := om[alias]
	if !ok {
		check.L("command not found", "command", alias)
		om["help"].Fn()
		os.Exit(2)
	}
	op.Fn()
}

func (om OPMap) Add(alias string, fn func()) {
	check.T(om[alias] == nil).P("alias in use", "alias", alias)
	om[alias] = &OP{Alias: alias, Name: "", Fn: fn}
}

func (om OPMap) help() {
	var ss []string
	for alias, op := range om {
		if op.Name == "" {
			ss = append(ss, alias)
		} else {
			ss = append(ss, fmt.Sprintf("%s => %s", alias, op.Name))
		}
	}
	slices.Sort(ss)
	fmt.Println(strings.Join(ss, "\n"))
}

func (om OPMap) symlink() {
	cleanOnly := flag.Bool("c", false, "clean only")
	resolveSymlink := flag.Bool("l", true, "resolve symlink of program")
	ParseFlag("prefix")
	prefix := flag.Arg(0)

	progName := check.V(filepath.Abs(os.Args[0])).F("filepath.abs", "arg0", os.Args[0])
	if *resolveSymlink {
		progName = ReadLastLink(progName)
	}
	binDir, binName := filepath.Split(progName)
	if wd := check.V(os.Getwd()).F("getwd"); wd != binDir {
		defer os.Chdir(wd)
		os.Chdir(binDir)
	}

	cmds, _ := filepath.Glob(fmt.Sprintf("%s.*", prefix))
	for _, c := range cmds {
		if IsSymlink(c) {
			os.Remove(c)
		}
	}
	if *cleanOnly {
		return
	}

	for _, op := range om {
		if op.Name != "" {
			name := fmt.Sprintf("%s.%s", prefix, op.Alias)
			check.L("create", "name", name, "op", op.Name)
			os.Symlink(binName, name)
		}
	}
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
	if _, ok := ops["help"]; !ok {
		ops["help"] = &OP{Alias: "help", Name: "Help", Fn: ops.help}
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
