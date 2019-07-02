package main

import (
	"bufio"
	"fmt"
	"math/bits"
	"os"
	"sort"
	"strings"
)

type IpV4 uint32

type Range struct {
	start IpV4
	end   IpV4
}

type OutputType int
type Ranges []Range
type IpPrinter func(IpV4)

const (
	otCidr  OutputType = 0
	otRange OutputType = 1
)

const MaxIp IpV4 = 0xFFFFFFFF

func max(a, b IpV4) IpV4 {
	if a < b {
		return b
	}
	return a
}

func maxInt(a, b int) int {
	if a < b {
		return b
	}
	return a
}

func printIp(t IpV4) {
	fmt.Printf("%d.%d.%d.%d", t>>24, t>>16&255, t>>8&255, t&255)
}

func toIp(a *[4]int) IpV4 {
	return IpV4(a[0]<<24 | a[1]<<16 | a[2]<<8 | a[3])
}

func printSingleIpAsRange(ip IpV4) {
	printIp(ip)
	fmt.Print("-")
	printIp(ip)
	fmt.Println()
}

func printAsRange(p *Range) {
	printIp(p.start)
	fmt.Print("-")
	printIp(p.end)
	fmt.Println()
}

func printSingleIpAsCidr(ip IpV4) {
	printIp(ip)
	fmt.Println("/32")
}

var printer IpPrinter = printIp

func printAsCidr(p *Range) {
	end := p.end
	s := p.start
	for {
		// assert(s <= end);
		// maybe overflow
		var size = uint32(end - s + 1)
		var cidr int
		if size != 0 {
			cidr = bits.LeadingZeros32(size) + 1
		} else {
			cidr = 0
		}
		if s != 0 {
			cidr = maxInt(cidr, 32-bits.TrailingZeros32(uint32(s)))
		}
		var e IpV4
		if cidr != 0 {
			e = s | ^(MaxIp << uint32(32-cidr))
		} else {
			e = MaxIp
		}
		printIp(s)
		fmt.Print("/")
		fmt.Printf("%d\n", cidr)
		if e >= end {
			break
		}
		s = e + 1
	}
}

func usage() {
	fmt.Println("usage")
}

func (s Ranges) Len() int { return len(s) }
func (s Ranges) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s Ranges) Less(i, j int) bool {
	if s[i].start < s[j].start {
		return true
	}
	//    if (s.pairs[i].start > s.pairs[j].start) { return false }
	//    if (s.pairs[i].end < s.pairs[j].end) { return true }
	return false
}

func main() {
	ot := otCidr
	output := printAsCidr
	var outputFile string
	force := false

	i, length := 1, len(os.Args)
	for ; i < length; i += 1 {
		switch arg := os.Args[i]; arg {
		case "--cidr":
			ot = otCidr
			continue
		case "--range":
			ot = otRange
			continue
		case "-o", "--output":
			i += 1
			if i < length {
				outputFile = os.Args[i]
			} else {
				usage()
				os.Exit(1)
			}
			continue
		case "--no-single":
			force = true
			continue
		case "--":
			i += 1
			break
		case "-?", "--help":
			usage()
			return
		default:
			if strings.HasPrefix(os.Args[i], "-") {
				fmt.Printf("unknown option '%s'\n", os.Args[i])
				usage()
				os.Exit(1)
			} else {
				break
			}
			continue
		}
		break
	}

	switch ot {
	case otCidr:
		output = printAsCidr
		if force {
			printer = printSingleIpAsCidr
		}
	case otRange:
		output = printAsRange
		if force {
			printer = printSingleIpAsRange
		}
	}

	if i < length {
		f, err := os.Open(os.Args[i])
		if err != nil {
			panic(err)
		}
		defer f.Close()

		oldStdin := os.Stdin
		defer func() { os.Stdin = oldStdin }()

		os.Stdin = f
	}
	if outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		oldStdout := os.Stdout
		defer func() { os.Stdout = oldStdout }()

		os.Stdout = f
	}

	var arr []Range
	input := bufio.NewScanner(os.Stdin)
	for input.Scan() {
		buf := input.Text()
		var a [4]int
		cnt, err := fmt.Sscanf(buf, "%d.%d.%d.%d", &a[0], &a[1], &a[2], &a[3])
		if err != nil {
			fmt.Fprintf(os.Stderr, "ignore line '%s'\n", buf)
			continue
		}
		start := toIp(&a)
		end := start
		cidr := 32

		cnt, err = fmt.Sscanf(buf, "%d.%d.%d.%d/%d", &a[0], &a[1], &a[2], &a[3], &cidr)
		if cnt == 5 {
			if cidr == 0 {
				start = 0
				end = MaxIp
			} else {
				start = IpV4(uint(start) & (0xFFFFFFFF << uint(32-cidr)))
				end = IpV4(uint(start) + (1 << uint(32-cidr)) - 1)
			}
		} else {
			cnt, err = fmt.Sscanf(buf, "%d.%d.%d.%d-%d.%d.%d.%d", &a[0], &a[1], &a[2], &a[3], &a[0], &a[1], &a[2], &a[3])
			if cnt == 8 {
				end = toIp(&a)
				if uint(end) < uint(start) {
					start, end = end, start
				}
			}
		}
		arr = append(arr, Range{
			start: start,
			end:   end,
		})
	}

	arrLen := len(arr)
	if arrLen > 0 {
		sort.Sort(Ranges(arr))
		j := 0
		for i := 1; i < arrLen; i += 1 {
			if arr[i].start <= arr[j].end+1 || arr[j].end == MaxIp {
				arr[j].end = max(arr[j].end, arr[i].end)
			} else {
				j += 1
				arr[j] = arr[i]
			}
		}
		arrLen = j + 1
	}
	for i := 0; i < arrLen; i += 1 {
		output(&arr[i])
	}
}
