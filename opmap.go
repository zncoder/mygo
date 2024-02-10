package mygo

import (
	"flag"
	"fmt"
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

type OPMap struct {
	ops     map[string]*OP
	binName string
}

func (om OPMap) Run(alias string) {
	op, ok := om.ops[alias]
	if !ok {
		check.L("command not found", "command", alias)
		om.ops["help"].Fn()
	}
	op.Fn()
}

func (om OPMap) Add(alias string, fn func()) {
	check.T(om.ops[alias] == nil).P("alias in use", "alias", alias)
	om.ops[alias] = &OP{Alias: alias, Name: "", Fn: fn}
}

func (om OPMap) RunCmd() {
	alias := filepath.Base(om.binName)
	i := strings.Index(alias, ".")
	if i < 0 {
		if len(os.Args) < 2 {
			om.ops["help"].Fn()
		}
		alias = os.Args[1]
		os.Args = os.Args[1:]
	} else {
		alias = alias[i+1:]
	}

	om.Run(alias)
}

func (om OPMap) help() {
	var ss []string
	for alias, op := range om.ops {
		if op.Name == "" {
			ss = append(ss, alias)
		} else {
			ss = append(ss, fmt.Sprintf("%s => %s", alias, op.Name))
		}
	}
	slices.Sort(ss)
	fmt.Println(strings.Join(ss, "\n"))
	os.Exit(2)
}

func (om OPMap) symlink() {
	cleanOnly := flag.Bool("c", false, "clean only")
	resolveSymlink := flag.Bool("l", true, "resolve symlink of program")
	ParseFlag("prefix")
	prefix := flag.Arg(0)

	progName := check.V(exec.LookPath(om.binName)).F("exec.lookpath", "arg0", om.binName)
	progName = check.V(filepath.Abs(progName)).F("filepath.abs", "progname", progName)
	if *resolveSymlink {
		progName = ReadLastLink(progName)
	}
	binDir, binName := filepath.Split(om.binName)
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

	for _, op := range om.ops {
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
	om := OPMap{
		ops:     make(map[string]*OP),
		binName: os.Args[0],
	}
	nameRe := regexp.MustCompile(`^([A-Z]+_)?([A-Z].*)$`)
	var op T
	rt := reflect.TypeOf(op)
	for i := 0; i < rt.NumMethod(); i++ {
		alias, name, fn := buildMethod[T](rt.Method(i), nameRe)
		_, ok := om.ops[alias]
		check.T(!ok).F("alias in use", "alias", alias)
		om.ops[alias] = &OP{Alias: alias, Name: name, Fn: func() { fn(op) }}
	}
	if _, ok := om.ops["help"]; !ok {
		om.ops["help"] = &OP{Alias: "help", Name: "Help", Fn: om.help}
	}
	if _, ok := om.ops["symlinkops"]; ok {
		check.F("symlinkops is use")
	}
	om.ops["symlinkops"] = &OP{Alias: "symlinkops", Name: "SymlinkOPs", Fn: om.symlink}
	return om
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
