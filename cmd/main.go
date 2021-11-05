package main

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/tools/godoc/util"

	goflags "github.com/jessevdk/go-flags"
)

type Flags struct {
	Context bool `short:"c" long:"help" description:"context (window) size"`
	Help    bool `short:"h" long:"help" description:"display this help"`
}

func main() {
	progName := "multigrep"
	if len(os.Args) > 0 {
		progName = os.Args[0]
	}
	usage := fmt.Sprintf("%s [options] <pattern> [[-v] <pattern> [...]]", progName)

	flags := Flags{}
	p := goflags.NewNamedParser("", goflags.PrintErrors|goflags.PassDoubleDash|goflags.PassAfterNonOption)
	p.AddGroup(usage, "", &flags)
	args, err := p.ParseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stdout, "failed to parse flags: %s\n", err)
		os.Exit(1)
	}
	if flags.Help {
		p.WriteHelp(os.Stdout)
		os.Exit(0)
	}

	numPatterns := len(args)
	if numPatterns < 1 {
		fmt.Fprintf(os.Stderr, "%s\n", usage)
		os.Exit(1)
	}

	var patterns []*regexp.Regexp
	for _, arg := range args {
		if arg == "-v" {
			panic("-v not yet supported")
		}
		r, err := regexp.Compile(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s is not a valid regexp: %s\n", arg, err.Error())
			os.Exit(1)
		}
		patterns = append(patterns, r)
	}

	dirPath := "."
	err = filepath.Walk(dirPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				if strings.HasPrefix(path, ".") && path != "." {
					return filepath.SkipDir
				}
				return nil
			}
			if (info.Mode() & fs.ModeSymlink) != 0 {
				fmt.Fprintf(os.Stderr, "Warn: skipping symlink %s\n", path)
				return nil
			}
			if strings.HasPrefix(path, ".") {
				return nil
			}
			return grepFile(path, patterns)
		})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to walk %s: %s\n", dirPath, err.Error())
		os.Exit(1)
	}
}

func grepFile(path string, regexps []*regexp.Regexp) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	if !util.IsText(data) {
		return nil
	}

	lines := strings.Split(string(data), "\n")
	numLines := len(lines)
	numRegexps := len(regexps)
	matches := make([]bool, numLines*numRegexps)
	for i, line := range lines {
		for j, reg := range regexps {
			k := i*numRegexps + j
			if reg.Match([]byte(line)) {
				matches[k] = true
			}
		}
	}

	// TODO make use of context size; for now just print the file if all patterns are found.

	var numUniqRegFound int
	regFound := make([]bool, numRegexps)
	for i := 0; i < numLines; i++ {
		for j := 0; j < numRegexps; j++ {
			k := i*numRegexps + j
			if matches[k] && !regFound[j] {
				regFound[j] = true
				numUniqRegFound++
			}
		}
	}
	if numUniqRegFound == numRegexps {
		fmt.Println(path)
	}

	return nil
}
