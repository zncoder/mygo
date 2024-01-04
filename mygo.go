package mygo

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
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
