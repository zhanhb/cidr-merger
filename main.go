package main

import (
	"bufio"
	"fmt"
	"github.com/pborman/getopt/v2"
	"io"
	"net"
	"os"
	"sort"
	"strings"
)

type Ranges []*Range

func (s Ranges) Len() int { return len(s) }
func (s Ranges) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s Ranges) Less(i, j int) bool {
	return lessThan(s[i].start, s[j].start)
}

//noinspection SpellCheckingInspection
func fprintln(w io.Writer, a ...interface{}) {
	if _, err := fmt.Fprintln(w, a...); err != nil {
		panic(err)
	}
}

func parseIp(str string) net.IP {
	ip := net.ParseIP(str)
	if ip != nil {
		if ipv4 := ip.To4(); ipv4 != nil {
			return ipv4
		}
	}
	return ip
}

// maybe IpCidr, Range or Ip is returned
func parse(text string) (IRange, error) {
	if index := strings.IndexByte(text, '/'); index != -1 {
		if _, network, err := net.ParseCIDR(text); err == nil {
			return IpNetWrapper{network}, nil
		} else {
			return nil, err
		}
	}
	if ip := parseIp(text); ip != nil {
		return IpWrapper{ip}, nil
	}
	if index := strings.IndexByte(text, '-'); index != -1 {
		start := parseIp(text[:index])
		end := parseIp(text[index+1:])
		if start != nil && end != nil {
			if len(start) == len(end) && !lessThan(end, start) {
				return &Range{start: start, end: end}, nil
			}
		}
		return nil, &net.ParseError{Type: "range", Text: text}
	}
	return nil, &net.ParseError{Type: "ip/CIDR address/range", Text: text}
}

func read(input *bufio.Scanner) []IRange {
	var arr []IRange
	for input.Scan() {
		if text := input.Text(); text != "" {
			maybe, err := parse(text)
			if err != nil {
				fprintln(os.Stderr, err)
			} else {
				arr = append(arr, maybe)
			}
		}
		if err := input.Err(); err != nil {
			panic(err)
		}
	}
	return arr
}

func readAll(inputFiles ...string) []IRange {
	var result []IRange
	for _, inputFile := range inputFiles {
		var input *bufio.Scanner
		if inputFile == "-" {
			input = bufio.NewScanner(os.Stdin)
		} else {
			in, err := os.Open(inputFile)
			if err != nil {
				panic(err)
			}
			//noinspection GoUnhandledErrorResult,GoDeferInLoop
			defer in.Close()
			input = bufio.NewScanner(in)
		}
		input.Split(bufio.ScanWords)
		result = append(result, read(input)...)
	}
	return result
}

func printAsIpNets(writer io.Writer, r IRange, simpler func(IRange) IRange) {
	for _, cidr := range r.ToIpNets() {
		fprintln(writer, simpler(IpNetWrapper{cidr}))
	}
}

func mainConsole(option *Option) {
	outputType := option.outputType
	simpler := option.simpler
	var printer func(writer io.Writer, r IRange)
	switch outputType {
	case OutputTypeRange:
		printer = func(writer io.Writer, r IRange) {
			fprintln(writer, simpler(r.ToRange()))
		}
	case OutputTypeCidr:
		printer = func(writer io.Writer, r IRange) {
			printAsIpNets(writer, r, simpler)
		}
	default:
		printer = func(writer io.Writer, r IRange) {
			switch r.(type) {
			case IpWrapper:
				fprintln(writer, r.ToIpNets()[0].String())
			case IpNetWrapper:
				fprintln(writer, simpler(r.ToRange()))
			case *Range:
				printAsIpNets(writer, r, simpler)
			default:
				panic("unreachable")
			}
		}
	}

	input := bufio.NewScanner(os.Stdin)
	input.Split(bufio.ScanWords)
	for input.Scan() {
		if text := input.Text(); text != "" {
			if r, err := parse(text); err != nil {
				fprintln(os.Stderr, err)
			} else {
				printer(os.Stdout, r)
			}
		}
	}
}

func process(option *Option, outputFile string, inputFiles ...string) {
	result := readAll(inputFiles...)
	if len(result) == 0 {
		switch option.emptyPolicy {
		case "error":
			panic("no data")
		case "skip":
			return
		default:
			// empty string if not specified, or "ignore"
		}
	}
	if option.originalOrder || len(result) < 2 {
		// noop
	} else {
		result = sortAndMerge(result)
	}
	result = convertBatch(result, option.simpler, option.outputType)
	var target *os.File
	if outputFile == "-" {
		target = os.Stdout
	} else if file, err := os.Create(outputFile); err != nil {
		panic(err)
	} else {
		//noinspection GoUnhandledErrorResult
		defer file.Close()
		target = file
	}
	writer := bufio.NewWriter(target)
	for _, r := range result {
		fprintln(writer, r)
	}
	if err := writer.Flush(); err != nil {
		panic(err)
	}
}

func convertBatch(wrappers []IRange, simpler func(IRange) IRange, outputType OutputType) []IRange {
	result := make([]IRange, 0, len(wrappers))
	if outputType == OutputTypeRange {
		for _, r := range wrappers {
			result = append(result, simpler(r.ToRange()))
		}
	} else {
		for _, r := range wrappers {
			for _, ipNet := range r.ToIpNets() {
				// can't use range iterator, for operator address of is taken
				// it seems a trick of golang here
				result = append(result, simpler(IpNetWrapper{ipNet}))
			}
		}
	}
	return result
}

func sortAndMerge(wrappers []IRange) []IRange {
	// assume len(wrappers) > 1
	ranges := make([]*Range, 0, len(wrappers))
	for _, e := range wrappers {
		ranges = append(ranges, e.ToRange())
	}
	sort.Sort(Ranges(ranges))

	res := make([]IRange, 0, len(ranges))
	now := ranges[0]
	familyLength := now.familyLength()
	start, end := now.start, now.end
	for i := 1; i < len(ranges); i++ {
		now := ranges[i]
		if fl := now.familyLength(); fl != familyLength {
			res = append(res, &Range{start, end})
			familyLength = fl
			start, end = now.start, now.end
			continue
		}
		if allFF(end) || !lessThan(addOne(end), now.start) {
			if lessThan(end, now.end) {
				end = now.end
			}
		} else {
			res = append(res, &Range{start, end})
			start, end = now.start, now.end
		}
	}
	return append(res, &Range{start, end})
}

func mainNormal(option *Option) {
	if outputSize := len(option.outputFiles); outputSize <= 1 {
		var outputFile string
		if outputSize == 1 {
			outputFile = option.outputFiles[0]
		} else {
			outputFile = "-"
		}
		process(option, outputFile, option.inputFiles...)
	} else if len(option.inputFiles) == outputSize {
		for i := 0; i < outputSize; i++ {
			process(option, option.outputFiles[i], option.inputFiles[i])
		}
	} else {
		panic("Input files' size doesn't match output files' size")
	}
}

func main() {
	defer func() {
		if err := recover(); err != nil {
			//noinspection GoUnhandledErrorResult
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}()

	getopt.HelpColumn = 28
	if option := parseOptions(); option.consoleMode {
		mainConsole(option)
	} else {
		mainNormal(option)
	}
}
