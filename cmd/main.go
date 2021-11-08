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
	Context         bool `short:"c" description:"context (window) size"`
	CaseInsensitive bool `short:"i" description:"Case insensitive matching"`
	WordBoundary    bool `short:"w" description:"word boundary matching"`
	Help            bool `short:"h" long:"help" description:"display this help"`
}

type pattern struct {
	re     *regexp.Regexp
	negate bool
}

func die(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg, args...)
	os.Exit(1)
}

func main() {
	progName := "multigrep"
	if len(os.Args) > 0 {
		progName = os.Args[0]
	}
	usage := fmt.Sprintf("%s [options] <pattern> [[-i] [-v] <pattern> [...]]", progName)

	flags := Flags{}
	p := goflags.NewNamedParser("", goflags.PrintErrors|goflags.PassDoubleDash|goflags.PassAfterNonOption)
	p.AddGroup(usage, "", &flags)
	args, err := p.ParseArgs(os.Args[1:])
	if err != nil {
		die("failed to parse flags: %s\n", err)
	}
	if flags.Help {
		p.WriteHelp(os.Stdout)
		os.Exit(0)
	}

	var patterns []*pattern
	negate := false
	insensitive := false
	ignoreDashes := false
	wordBoundary := false
	for _, arg := range args {
		if len(arg) == 0 {
			die("empty args are not supported\n")
		}
		if arg[0] == '-' && !ignoreDashes {
			for _, short := range arg[1:] {
				switch short {
				case 'i':
					if insensitive {
						die("two -i's in a row not supported\n")
					}
					insensitive = true
				case 'v':
					if negate {
						die("two -v's in a row not supported\n")
					}
					negate = true
				case 'w':
					if wordBoundary {
						die("two -w's in a row not supported\n")
					}
					wordBoundary = true
				case 'e':
					ignoreDashes = true
				default:
					die("Error: -%c not recognized\n", short)
				}
			}
			continue
		}
		if flags.WordBoundary || wordBoundary {
			arg = "\\b" + arg + "\\b"
		}
		if flags.CaseInsensitive || insensitive {
			arg = "(?i)" + arg
		}
		r, err := regexp.Compile(arg)
		if err != nil {
			die("%s is not a valid regexp: %s\n", arg, err.Error())
		}
		patterns = append(patterns, &pattern{
			re:     r,
			negate: negate,
		})
		negate = false
		insensitive = false
		wordBoundary = false
		ignoreDashes = false
	}

	numPatterns := len(patterns)
	if numPatterns < 1 {
		die("%s\n", usage)
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
		die("failed to walk %s: %s\n", dirPath, err.Error())
	}
}

func grepFile(path string, patterns []*pattern) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	if !util.IsText(data) {
		return nil
	}

	lines := strings.Split(string(data), "\n")
	numLines := len(lines)
	numRegexps := len(patterns)
	matches := make([]bool, numLines*numRegexps)
	for i, line := range lines {
		for j, pat := range patterns {
			k := i*numRegexps + j
			match := pat.re.Match([]byte(line))
			matches[k] = match != pat.negate // xor
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
