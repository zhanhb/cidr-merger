package main

import (
	"io"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/pborman/getopt/v2"
)

type OutputType byte

type ConsoleMode byte

type Option struct {
	inputFiles    []string
	outputFiles   []string
	simpler       func(IRange) IRange
	emptyPolicy   string
	outputType    OutputType
	consoleMode   bool
	originalOrder bool
}

const (
	OutputTypeCidr OutputType = iota + 1
	OutputTypeRange
	OutputTypeSum = OutputTypeCidr + OutputTypeRange
)

const (
	ConsoleModeNotSpecified ConsoleMode = iota
	ConsoleModeBatch
	ConsoleModeConsole
	ConsoleModeSum = ConsoleModeBatch + ConsoleModeConsole
)

func printUsage(set *getopt.Set, file io.Writer, extra ...interface{}) {
	if len(extra) > 0 {
		fprintln(file, extra...)
	}
	for _, r := range []interface{}{
		strings.Join([]string{
			"Usage:",
			set.Program(),
			"[OPTION]... [FILE]...",
		}, " "),
		"Write sorted result to standard output.",
		"",
		"Options:",
	} {
		fprintln(file, r)
	}
	set.PrintOptions(file)
}

func tty(file *os.File) bool {
	return isatty.IsTerminal(file.Fd()) || isatty.IsCygwinTerminal(file.Fd())
}

func parseOptions() *Option {
	var (
		sharedBool  bool
		outputFile  string
		outputType  OutputType
		consoleMode ConsoleMode
	)

	options := getopt.New()
	batchModeValue := options.FlagLong(&sharedBool, "batch", 0, "batch mode (default if input files supplied or stdin is not a tty), read file content into memory, then write to the specified file").Value()
	consoleModeValue := options.FlagLong(&sharedBool, "console", 'c', "console mode(default if no input files supplied and stdin is a tty), all input output files are ignored, write to stdout immediately").Value()
	outputAsCidr := options.FlagLong(&sharedBool, "cidr", 0, "print as ip/cidr (default if not console mode)").Value()
	outputAsRange := options.FlagLong(&sharedBool, "range", 'r', "print as ip ranges").Value()
	emptyPolicy := options.EnumLong("empty-policy", 0,
		[]string{"ignore", "skip", "error"}, "",
		"indicate how to process empty input file\n  ignore(default): process as if it is not empty\n  skip: don't create output file\n  error: raise an error and exit")
	outputFileValue := options.FlagLong(&outputFile, "output", 'o', "output values to <file>, if multiple output files specified, the count should be same as input files, and will be processed respectively", "file").Value()
	errorEmpty := options.FlagLong(&sharedBool, "error-if-empty", 'e', "same as --empty-policy=error").Value()
	skipEmpty := options.FlagLong(&sharedBool, "skip-empty", 'k', "same as --empty-policy=skip").Value()
	ignoreEmpty := options.FlagLong(&sharedBool, "ignore-empty", 0, "same as --empty-policy=ignore").Value()
	simple := options.FlagLong(&sharedBool, "simple", 0, "output as single ip as possible (default)\n  ie. 192.168.1.2/32 -> 192.168.1.2\n      192.168.1.2-192.168.1.2 -> 192.168.1.2").Value()
	standard := options.BoolLong("standard", 's', "don't output as single ip")
	merge := options.FlagLong(&sharedBool, "merge", 0, "sort and merge input values (default)").Value()
	originalOrder := options.BoolLong("original-order", 0, "output as the order of input, without merging")
	help := options.FlagLong(&sharedBool, "help", 'h', "show this help menu").Value()
	version := options.FlagLong(&sharedBool, "version", 'v', "show version info").Value()

	reverse := map[getopt.Value]*bool{
		simple: standard,
		merge:  originalOrder,
	}

	policyDelegate := map[getopt.Value]string{
		errorEmpty:  "error",
		skipEmpty:   "skip",
		ignoreEmpty: "ignore",
	}

	outputMap := map[getopt.Value]OutputType{
		outputAsCidr:  OutputTypeCidr,
		outputAsRange: OutputTypeRange,
	}

	consoleModeMap := map[getopt.Value]ConsoleMode{
		batchModeValue:   ConsoleModeBatch,
		consoleModeValue: ConsoleModeConsole,
	}

	var outputFiles []string

	customAction := map[getopt.Value]func() bool{
		help: func() bool {
			printUsage(options, os.Stdout)
			return false
		}, version: func() bool {
			println("cidr merger " + VERSION)
			return false
		}, outputFileValue: func() bool {
			outputFiles = append(outputFiles, outputFile)
			return true
		},
	}
	if err := options.Getopt(os.Args, func(opt getopt.Option) bool {
		value := opt.Value()
		if k, ok := reverse[value]; ok {
			*k = !sharedBool
			return true
		} else if k, ok := policyDelegate[value]; ok {
			if sharedBool {
				*emptyPolicy = k
			} else {
				*emptyPolicy = ""
			}
			return true
		} else if k, ok := outputMap[value]; ok {
			if sharedBool {
				outputType = k
			} else {
				outputType = OutputTypeSum - k
			}
			return true
		} else if k, ok := consoleModeMap[value]; ok {
			if sharedBool {
				consoleMode = k
			} else {
				consoleMode = ConsoleModeSum - k
			}
			return true
		} else if k, ok := customAction[value]; ok {
			return k()
		}
		return opt.Seen()
	}); err != nil {
		printUsage(options, os.Stderr, err)
		os.Exit(1)
	}

	if options.State() == getopt.Terminated {
		os.Exit(0)
	}

	inputFiles := options.Args()
	if consoleMode == ConsoleModeNotSpecified {
		if tty(os.Stdin) && len(inputFiles) == 0 && len(outputFiles) == 0 {
			consoleMode = ConsoleModeConsole
		} else {
			consoleMode = ConsoleModeBatch
		}
	}
	simpler := singleOrSelf
	if *standard {
		simpler = returnSelf
	}

	return &Option{
		inputFiles:    inputFiles,
		outputFiles:   outputFiles,
		outputType:    outputType,
		consoleMode:   consoleMode == ConsoleModeConsole,
		simpler:       simpler,
		originalOrder: *originalOrder,
		emptyPolicy:   *emptyPolicy,
	}
}
