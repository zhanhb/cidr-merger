package main

import (
	"bufio"
	"fmt"
	"github.com/pborman/getopt/v2"
	"math/bits"
	"net"
	"os"
	"sort"
	"strings"
)

type Range struct {
	start net.IP
	end   net.IP
}

type Wrapper interface {
	String(simple bool) string
	toIpNets() []net.IPNet
	toRange() Range
}

func (r Range) familyLength() int {
	return len(r.start)
}
func (r Range) String(simple bool) string {
	if simple && r.start.Equal(r.end) {
		return r.start.String()
	}
	return r.start.String() + "-" + r.end.String()
}
func (r Range) toIpNets() []net.IPNet {
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
		var size = addOne(minus(end, s))
		cidr := max(leadingZero(size)+1, ipBits-trailingZeros(s))
		mask := net.CIDRMask(cidr, ipBits)
		if len(mask)*8 != ipBits {
			panic("assert failed: " + s.String() + " " + mask.String())
		}
		ipNet := net.IPNet{IP: s, Mask: mask}
		tmp := lastIp(&ipNet)
		result = append(result, ipNet)
		if !lessThan(tmp, end) {
			return result
		}
		s = addOne(tmp)
		isAllZero = false
	}
}
func (r Range) toRange() Range {
	return r
}

type IpWrapper struct {
	value net.IP
}

func (r IpWrapper) String(bool) string {
	return r.value.String()
}
func (r IpWrapper) toIpNets() []net.IPNet {
	ipBits := len(r.value) * 8
	return []net.IPNet{
		{IP: r.value, Mask: net.CIDRMask(ipBits, ipBits)},
	}
}
func (r IpWrapper) toRange() Range {
	return Range{start: r.value, end: r.value}
}

type IpNetWrapper struct {
	value *net.IPNet
}

func (r IpNetWrapper) String(simple bool) string {
	if ones, bts := r.value.Mask.Size(); simple && ones == bts {
		return r.value.IP.String()
	}
	return r.value.String()
}
func (r IpNetWrapper) toIpNets() []net.IPNet {
	return []net.IPNet{*r.value}
}
func (r IpNetWrapper) toRange() Range {
	ipNet := r.value
	return Range{start: ipNet.IP, end: lastIp(ipNet)}
}

type Ranges []Range

func lessThan(a, b net.IP) bool {
	ipLen := len(a)
	for i := 0; i < ipLen; i++ {
		if a[i] < b[i] {
			return true
		}
		if a[i] > b[i] {
			return false
		}
	}
	return false
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
		c := ip[i]
		if c != 0 {
			return (ipLen-i-1)*8 + bits.TrailingZeros8(c)
		}
	}
	return ipLen * 8
}

func lastIp(ipNet *net.IPNet) net.IP {
	ipLen := len(ipNet.IP)
	res := make(net.IP, ipLen)
	mask := ipNet.Mask
	if len(mask) != ipLen {
		panic("assert failed: unexpected IPNet " + ipNet.String())
	}
	for i := 0; i < ipLen; i++ {
		res[i] = ipNet.IP[i] | ^mask[i]
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
		panic("assert failed: unexpected ip " + ip.String())
	}
	return to
}

func minus(a, b net.IP) net.IP {
	ipLen := len(a)
	var result net.IP = make([]byte, ipLen)
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
		panic("assert failed: subtract " + b.String() + " from " + a.String())
	}
	return result
}

func (s Ranges) Len() int { return len(s) }
func (s Ranges) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s Ranges) Less(i, j int) bool {
	si, sj := s[i].start, s[j].start
	lenOfI := len(si)
	lenOfJ := len(sj)
	if lenOfI < lenOfJ {
		return true
	} else if lenOfI > lenOfJ {
		return false
	}
	return lessThan(si, sj)
}

type OutputType int

type Option struct {
	inputFiles    []string
	outputFiles   []string
	outputType    OutputType
	consoleMode   bool
	standard      bool
	originalOrder bool
	emptyPolicy   string
}

const (
	OutputTypeNotSpecified OutputType = iota
	OutputTypeDefault
	OutputTypeCidr
	OutputTypeRange
)

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
	options.SetParameters("[files ...]")

	reverse := make(map[getopt.Value]*bool)
	reverse[batchModeValue] = consoleMode
	reverse[simple] = standard
	reverse[merge] = originalOrder

	policyDelegate := make(map[getopt.Value]string)
	policyDelegate[errorEmpty] = "error"
	policyDelegate[skipEmpty] = "skip"
	policyDelegate[ignoreEmpty] = "ignore"

	outputMap := make(map[getopt.Value]OutputType)
	outputMap[outputAsCidr] = OutputTypeCidr
	outputMap[outputAsRange] = OutputTypeRange

	var outputFiles []string

	customAction := make(map[getopt.Value]func() bool)
	customAction[help] = func() bool {
		parts := make([]string, 3, 4)
		parts[0] = "Usage:"
		parts[1] = options.Program()
		parts[2] = "[Options]"
		if params := options.Parameters(); params != "" {
			parts = append(parts, params)
		}
		w := os.Stdout
		_, err := fmt.Fprintln(w, strings.Join(parts, " "))
		if err != nil {
			panic(err)
		}
		options.PrintOptions(w)
		return false
	}
	customAction[version] = func() bool {
		println("cidr merger 0.1")
		return false
	}
	customAction[outputFileValue] = func() bool {
		outputFiles = append(outputFiles, outputFile)
		return true
	}
	if err := options.Getopt(os.Args, func(opt getopt.Option) bool {
		value := opt.Value()
		if k := reverse[value]; k != nil {
			*k = !dummy
			return true
		} else if k := policyDelegate[value]; k != "" {
			*emptyPolicy = k
			return true
		} else if k := outputMap[value]; k != OutputTypeNotSpecified {
			outputType = k
			return true
		} else if k := customAction[value]; k != nil {
			return k()
		}
		return opt.Seen()
	}); err != nil {
		_, err = fmt.Fprintln(os.Stderr, err)
		if err != nil {
			panic(err)
		}
		options.PrintUsage(os.Stderr)
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
	if *emptyPolicy == "" {
		*emptyPolicy = "ignore"
	}
	return Option{
		inputFiles:    inputFiles,
		outputFiles:   outputFiles,
		outputType:    outputType,
		consoleMode:   *consoleMode,
		standard:      *standard,
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
func parse(line string) (Wrapper, error) {
	if _, network, err := net.ParseCIDR(line); err == nil {
		return IpNetWrapper{network}, nil
	}
	if ip := parseIp(line); ip != nil {
		return IpWrapper{ip}, nil
	}
	if index := strings.IndexByte(line, '-'); index != -1 {
		start := parseIp(line[:index])
		end := parseIp(line[index+1:])
		if len(start) == len(end) && start != nil {
			if lessThan(end, start) {
				return nil, &net.ParseError{Type: "range", Text: line}
			}
			return Range{start: start, end: end}, nil
		}
	}
	return nil, &net.ParseError{Type: "ip/cidr/range", Text: line}
}

func read(input *bufio.Scanner) []Wrapper {
	var arr []Wrapper
	for input.Scan() {
		text := input.Text()
		if text != "" {
			maybe, err := parse(text)
			if err != nil {
				_, err = fmt.Fprintf(os.Stderr, "%v\n", err)
				if err != nil {
					panic(err)
				}
				continue
			}
			arr = append(arr, maybe)
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
		result = append(result, read(input)...)
	}
	return result
}

func mainConsole(option *Option) {
	doAsCidr := func(writer func(string), r Wrapper, simple bool) {
		for _, cidr := range r.toIpNets() {
			writer(IpNetWrapper{&cidr}.String(simple))
		}
	}

	simple, outputType := !option.standard, option.outputType
	var printer func(writer func(string), r Wrapper)
	switch outputType {
	case OutputTypeRange:
		printer = func(writer func(string), r Wrapper) {
			writer(r.toRange().String(simple))
		}
	case OutputTypeCidr:
		printer = func(writer func(string), r Wrapper) {
			doAsCidr(writer, r, simple)
		}
	default:
		printer = func(writer func(string), r Wrapper) {
			switch r.(type) {
			case IpWrapper:
				doAsCidr(writer, r, false)
			case IpNetWrapper:
				writer(r.toRange().String(simple))
			case Range:
				doAsCidr(writer, r, simple)
			default:
				panic("should not reached")
			}
		}
	}

	input := bufio.NewScanner(os.Stdin)
	for ; input.Scan(); {
		text := input.Text()
		if text != "" {
			r, err := parse(text)
			if err != nil {
				_, err = os.Stderr.WriteString(fmt.Sprintf("%v", err) + "\n")
				if err != nil {
					panic(err)
				}
			} else {
				printer(func(s string) {
					_, err = os.Stdout.WriteString(s + "\n")
					if err != nil {
						panic(err)
					}
				}, r)
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
		}
	}
	arrLen := len(result)
	if option.originalOrder || arrLen < 2 {
		// noop
	} else {
		var ranges []Range
		for _, e := range result {
			ranges = append(ranges, e.toRange())
		}
		sort.Sort(Ranges(ranges))

		var res []Wrapper
		now := ranges[0]
		familyLength := now.familyLength()
		start, end := now.start, now.end
		for i := 1; i < arrLen; i++ {
			now := ranges[i]
			if fl := now.familyLength(); fl != familyLength {
				res = append(res, Range{start, end})
				familyLength = fl
				start, end = now.start, now.end
				continue
			}
			if allFF(end) || !lessThan(addOne(end), now.start) {
				if lessThan(end, now.end) {
					end = now.end
				}
			} else {
				res = append(res, Range{start, end})
				start, end = now.start, now.end
			}
		}
		res = append(res, Range{start, end})
		result = res
	}
	var target *os.File
	if outputFile == "-" {
		target = os.Stdout
	} else {
		file, err := os.Create(outputFile)
		if err != nil {
			panic(err)
		}
		//noinspection GoUnhandledErrorResult,GoDeferInLoop
		defer file.Close()
		target = file
	}
	writer := bufio.NewWriter(target)
	simple := !option.standard
	for _, r := range result {
		if option.outputType == OutputTypeRange {
			_, err := writer.WriteString(r.toRange().String(simple) + "\n")
			if err != nil {
				panic(err)
			}
		} else {
			for _, ipNet := range r.toIpNets() {
				_, err := writer.WriteString(IpNetWrapper{&ipNet}.String(simple) + "\n")
				if err != nil {
					panic(err)
				}
			}
		}
	}
	err := writer.Flush()
	if err != nil {
		panic(err)
	}
}

func mainNormal(option *Option) {
	outputSize := len(option.outputFiles)
	if outputSize == 0 || outputSize == 1 {
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
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(2)
		}
	}()

	getopt.HelpColumn = 28
	option := parseOptions()

	if option.consoleMode {
		mainConsole(&option)
	} else {
		mainNormal(&option)
	}
}
