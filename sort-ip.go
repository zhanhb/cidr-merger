package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/pborman/getopt/v2"
	"io"
	"math/bits"
	"net"
	"os"
	"sort"
	"strings"
)

type Wrapper interface {
	ToIp() net.IP // return nil if can't be represented as a single ip
	ToIpNets() []net.IPNet
	ToRange() *Range
	String() string
}

type Range struct {
	start net.IP
	end   net.IP
}

func (r *Range) familyLength() int {
	return len(r.start)
}
func (r *Range) ToIp() net.IP {
	if bytes.Equal(r.start, r.end) {
		return r.start
	}
	return nil
}
func (r *Range) ToIpNets() []net.IPNet {
	end := r.end
	s := r.start
	ipBits := len(s) * 8
	isAllZero := allZero(s)
	if isAllZero && allFF(end) {
		return []net.IPNet{
			{IP: s, Mask: net.CIDRMask(0, ipBits)},
		}
	}
	var result []net.IPNet
	for {
		// assert s <= end;
		// will never overflow
		cidr := max(leadingZero(addOne(minus(end, s)))+1, ipBits-trailingZeros(s))
		ipNet := net.IPNet{IP: s, Mask: net.CIDRMask(cidr, ipBits)}
		result = append(result, ipNet)
		tmp := lastIp(&ipNet)
		if !lessThan(tmp, end) {
			return result
		}
		s = addOne(tmp)
		isAllZero = false
	}
}
func (r *Range) ToRange() *Range {
	return r
}
func (r *Range) String() string {
	return r.start.String() + "-" + r.end.String()
}

type IpWrapper struct {
	net.IP
}

func (r *IpWrapper) ToIp() net.IP {
	return r.IP
}
func (r *IpWrapper) ToIpNets() []net.IPNet {
	ipBits := len(r.IP) * 8
	return []net.IPNet{
		{IP: r.IP, Mask: net.CIDRMask(ipBits, ipBits)},
	}
}
func (r *IpWrapper) ToRange() *Range {
	return &Range{start: r.IP, end: r.IP}
}

type IpNetWrapper struct {
	net.IPNet
}

func (r *IpNetWrapper) ToIp() net.IP {
	if ones, bts := r.IPNet.Mask.Size(); ones == bts {
		return r.IPNet.IP
	}
	return nil
}
func (r *IpNetWrapper) ToIpNets() []net.IPNet {
	return []net.IPNet{r.IPNet}
}
func (r *IpNetWrapper) ToRange() *Range {
	ipNet := r.IPNet
	return &Range{start: ipNet.IP, end: lastIp(&ipNet)}
}

func lessThan(a, b net.IP) bool {
	if lenA, lenB := len(a), len(b); lenA != lenB {
		return lenA < lenB
	}
	return bytes.Compare(a, b) < 0
}

func max(a, b int) int {
	if a < b {
		return b
	}
	return a
}

func allFF(ip net.IP) bool {
	for _, c := range ip {
		if c != 0xff {
			return false
		}
	}
	return true
}

func allZero(ip net.IP) bool {
	for _, c := range ip {
		if c != 0 {
			return false
		}
	}
	return true
}

func leadingZero(ip net.IP) int {
	for index, c := range ip {
		if c != 0 {
			return index*8 + bits.LeadingZeros8(c)
		}
	}
	return len(ip) * 8
}

func trailingZeros(ip net.IP) int {
	ipLen := len(ip)
	for i := ipLen - 1; i >= 0; i-- {
		if c := ip[i]; c != 0 {
			return (ipLen-i-1)*8 + bits.TrailingZeros8(c)
		}
	}
	return ipLen * 8
}

func lastIp(ipNet *net.IPNet) net.IP {
	ip, mask := ipNet.IP, ipNet.Mask
	ipLen := len(ip)
	res := make(net.IP, ipLen)
	if len(mask) != ipLen {
		panic("unreachable: unexpected IPNet " + ipNet.String())
	}
	for i := 0; i < ipLen; i++ {
		res[i] = ip[i] | ^mask[i]
	}
	return res
}

func addOne(ip net.IP) net.IP {
	ipLen := len(ip)
	to := make(net.IP, ipLen)
	var add byte = 1
	for i := ipLen - 1; i >= 0; i-- {
		res := ip[i] + add
		to[i] = res
		if res != 0 {
			add = 0
		}
	}
	if add != 0 {
		panic("unreachable: unexpected ip " + ip.String())
	}
	return to
}

func minus(a, b net.IP) net.IP {
	ipLen := len(a)
	result := make(net.IP, ipLen)
	var borrow byte = 0
	for i := ipLen - 1; i >= 0; i-- {
		result[i] = a[i] - b[i] - borrow
		if result[i] > a[i] {
			borrow = 1
		} else {
			borrow = 0
		}
	}
	if borrow != 0 {
		panic("unreachable: subtract " + b.String() + " from " + a.String())
	}
	return result
}

//noinspection SpellCheckingInspection
func fprintln(w io.Writer, a ...interface{}) {
	if _, err := fmt.Fprintln(w, a...); err != nil {
		panic(err)
	}
}

type Ranges []*Range

func (s Ranges) Len() int { return len(s) }
func (s Ranges) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s Ranges) Less(i, j int) bool {
	return lessThan(s[i].start, s[j].start)
}

type OutputType byte

type Option struct {
	inputFiles    []string
	outputFiles   []string
	simpler       func(Wrapper) Wrapper
	emptyPolicy   string
	outputType    OutputType
	consoleMode   bool
	originalOrder bool
}

const (
	OutputTypeNotSpecified OutputType = iota
	OutputTypeDefault
	OutputTypeCidr
	OutputTypeRange
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

func parseOptions() Option {
	var (
		dummy      bool
		outputFile string
		outputType = OutputTypeDefault
	)

	options := getopt.New()
	batchModeValue := options.FlagLong(&dummy, "batch", 0, "batch mode (default), read file content into memory, then write to the specified file").Value()
	consoleMode := options.BoolLong("console", 'c', "console mode, all input output files are ignored, write to stdout immediately")
	outputAsCidr := options.FlagLong(&dummy, "cidr", 0, "print as ip/cidr (default if not console mode)").Value()
	outputAsRange := options.FlagLong(&dummy, "range", 'r', "print as ip ranges").Value()
	emptyPolicy := options.EnumLong("empty-policy", 0,
		[]string{"ignore", "skip", "error"}, "",
		"indicate how to process empty input file\n  ignore(default): process as if it is not empty\n  skip: don't create output file\n  error: raise an error and exit")
	outputFileValue := options.FlagLong(&outputFile, "output", 'o', "output values to <file>, if multiple output files specified, the count should be same as input files, and will be processed respectively", "file").Value()
	errorEmpty := options.FlagLong(&dummy, "error-if-empty", 'e', "same as --empty-policy=error").Value()
	skipEmpty := options.FlagLong(&dummy, "skip-empty", 'k', "same as --empty-policy=skip").Value()
	ignoreEmpty := options.FlagLong(&dummy, "ignore-empty", 0, "same as --empty-policy=ignore").Value()
	simple := options.FlagLong(&dummy, "simple", 0, "output as single ip as possible (default)\n  ie. 192.168.1.2/32 -> 192.168.1.2\n      192.168.1.2-192.168.1.2 -> 192.168.1.2").Value()
	standard := options.BoolLong("standard", 's', "don't output as single ip")
	merge := options.FlagLong(&dummy, "merge", 0, "sort and merge input values (default)").Value()
	originalOrder := options.BoolLong("original-order", 0, "output as the order of input, without merging")
	help := options.FlagLong(&dummy, "help", 'h', "show this help menu").Value()
	version := options.FlagLong(&dummy, "version", 'v', "show version info").Value()

	reverse := map[getopt.Value]*bool{
		batchModeValue: consoleMode,
		simple:         standard,
		merge:          originalOrder,
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

	var outputFiles []string

	customAction := map[getopt.Value]func() bool{
		help: func() bool {
			printUsage(options, os.Stdout)
			return false
		}, version: func() bool {
			println("cidr merger 0.1")
			return false
		}, outputFileValue: func() bool {
			outputFiles = append(outputFiles, outputFile)
			return true
		},
	}
	if err := options.Getopt(os.Args, func(opt getopt.Option) bool {
		value := opt.Value()
		if k := reverse[value]; k != nil {
			*k = !dummy
			return true
		} else if k := policyDelegate[value]; k != "" {
			if dummy {
				*emptyPolicy = k
			} else {
				*emptyPolicy = ""
			}
			return true
		} else if k := outputMap[value]; k != OutputTypeNotSpecified {
			outputType = k
			return true
		} else if k := customAction[value]; k != nil {
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

	var inputFiles []string
	if args := options.Args(); len(args) == 0 {
		inputFiles = []string{"-"}
	} else {
		inputFiles = args
	}
	simpler := func(r Wrapper) Wrapper {
		if ip := r.ToIp(); ip != nil {
			return &IpWrapper{ip}
		}
		return r
	}
	if *standard {
		simpler = func(r Wrapper) Wrapper {
			return r
		}
	}
	return Option{
		inputFiles:    inputFiles,
		outputFiles:   outputFiles,
		outputType:    outputType,
		consoleMode:   *consoleMode,
		simpler:       simpler,
		originalOrder: *originalOrder,
		emptyPolicy:   *emptyPolicy,
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
func parse(text string) (Wrapper, error) {
	if _, network, err := net.ParseCIDR(text); err == nil {
		return &IpNetWrapper{*network}, nil
	}
	if ip := parseIp(text); ip != nil {
		return &IpWrapper{ip}, nil
	}
	if index := strings.IndexByte(text, '-'); index != -1 {
		start := parseIp(text[:index])
		end := parseIp(text[index+1:])
		if len(start) == len(end) && start != nil {
			if lessThan(end, start) {
				return nil, &net.ParseError{Type: "range", Text: text}
			}
			return &Range{start: start, end: end}, nil
		}
	}
	return nil, &net.ParseError{Type: "ip/cidr/range", Text: text}
}

func read(input *bufio.Scanner) []Wrapper {
	var arr []Wrapper
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

func readAll(inputFiles ...string) []Wrapper {
	var result []Wrapper
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

func printAsIpNets(writer io.Writer, r Wrapper, simpler func(Wrapper) Wrapper) {
	for _, cidr := range r.ToIpNets() {
		fprintln(writer, simpler(&IpNetWrapper{cidr}))
	}
}

func mainConsole(option *Option) {
	outputType := option.outputType
	simpler := option.simpler
	var printer func(writer io.Writer, r Wrapper)
	switch outputType {
	case OutputTypeRange:
		printer = func(writer io.Writer, r Wrapper) {
			fprintln(writer, simpler(r.ToRange()))
		}
	case OutputTypeCidr:
		printer = func(writer io.Writer, r Wrapper) {
			printAsIpNets(writer, r, simpler)
		}
	default:
		printer = func(writer io.Writer, r Wrapper) {
			switch r.(type) {
			case *IpWrapper:
				fprintln(writer, r.ToIpNets()[0].String())
			case *IpNetWrapper:
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

func convertBatch(wrappers []Wrapper, simpler func(Wrapper) Wrapper, outputType OutputType) []Wrapper {
	result := make([]Wrapper, 0, len(wrappers))
	if outputType == OutputTypeRange {
		for _, r := range wrappers {
			result = append(result, simpler(r.ToRange()))
		}
	} else {
		for _, r := range wrappers {
			for _, ipNet := range r.ToIpNets() {
				// can't use range iterator, for operator address of is taken
				// it seems a trick of golang here
				result = append(result, simpler(&IpNetWrapper{ipNet}))
			}
		}
	}
	return result
}

func sortAndMerge(wrappers []Wrapper) []Wrapper {
	// assume len(wrappers) > 1
	ranges := make([]*Range, 0, len(wrappers))
	for _, e := range wrappers {
		ranges = append(ranges, e.ToRange())
	}
	sort.Sort(Ranges(ranges))

	res := make([]Wrapper, 0, len(ranges))
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
		mainConsole(&option)
	} else {
		mainNormal(&option)
	}
}
