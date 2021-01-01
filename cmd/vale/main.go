package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/errata-ai/vale/v2/internal/core"
	"github.com/errata-ai/vale/v2/internal/lint"
)

// version is set during the release build process.
var version = "master"

func validateFlags(cfg *core.Config) error {
	if cfg.Flags.Path != "" && !core.FileExists(cfg.Flags.Path) {
		return core.NewE100(
			"--config",
			fmt.Errorf("path '%s' does not exist", cfg.Flags.Path))
	}
	return nil
}

func stat() bool {
	stat, err := os.Stdin.Stat()
	if err != nil || (stat.Mode()&os.ModeCharDevice) != 0 {
		return false
	}
	return true
}

func looksLikeStdin(s string) bool {
	return !(core.FileExists(s) || core.IsDir(s)) && s != ""
}

func doLint(args []string, l *lint.Linter, glob string) ([]*core.File, error) {
	var linted []*core.File
	var err error

	length := len(args)
	if length > 0 {
		if length == 1 && looksLikeStdin(args[0]) {
			// Case 1:
			//
			// $ vale "some text in a string"
			linted, err = l.LintString(args[0])
		} else {
			// Case 2:
			//
			// $ vale file1 dir1 file2
			input := []string{}
			for _, file := range args {
				if looksLikeStdin(file) {
					return linted, core.NewE100(
						"doLint",
						fmt.Errorf("argument '%s' does not exist", file),
					)
				}
				input = append(input, file)
			}
			linted, err = l.Lint(input, glob)
		}
	} else {
		// Case 3:
		//
		// $ cat file.md | vale
		stdin, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return linted, core.NewE100("doLint", err)
		}
		linted, err = l.LintString(string(stdin))
	}

	return linted, err
}

func handleError(err error) {
	ShowError(err, flags.Output, os.Stderr)
	os.Exit(2)
}

func main() {
	flag.Parse()

	config, err := core.NewConfig(&flags)
	if err != nil {
		ShowError(err, flags.Output, os.Stderr)
	}

	args := flag.Args()
	if len(args) == 0 && !stat() {
		printIntro()
	} else if len(args) == 1 && core.StringInSlice(args[0], commands) {
		if err = doCommand(args[0]); err != nil {
			os.Exit(2)
		} else {
			os.Exit(1)
		}
	} else if err := validateFlags(config); err != nil {
		handleError(err)
	} else if err = core.From("ini", config); err != nil {
		handleError(err)
	}

	linter, err := lint.NewLinter(config)
	if err != nil {
		handleError(err)
	}

	linted, err := doLint(args, linter, flags.Glob)
	if err != nil {
		handleError(err)
	}

	hasErrors, err := PrintAlerts(linted, config)
	if err != nil {
		handleError(err)
	} else if hasErrors && !flags.NoExit {
		os.Exit(1)
	}

	os.Exit(0)
}
