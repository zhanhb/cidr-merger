package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
)

type IP uint32

type Range struct {
	start IP;
	end   IP;
};

type OutputType int
type Ranges []Range
type IpPrinter func(IP)

const (
	ot_cidr  OutputType = iota
	ot_range
)

const MAX_IP IP = 0xFFFFFFFF

func __builtin_ctz(u uint32) int {
	if u == 0 {
		return 32
	}
	var y uint32
	n := 31
	y = u << 16;
	if (y != 0) {
		n = n - 16;
		u = y;
	}
	y = u << 8;
	if (y != 0) {
		n = n - 8;
		u = y;
	}
	y = u << 4;
	if (y != 0) {
		n = n - 4;
		u = y;
	}
	y = u << 2;
	if (y != 0) {
		n = n - 2;
		u = y;
	}
	return n - int((u<<1)>>31)
}

func __builtin_clz(i uint32) int {
	if i == 0 {
		return 32
	}
	n := 1
	if (i>>16 == 0) {
		n += 16;
		i <<= 16;
	}
	if (i>>24 == 0) {
		n += 8;
		i <<= 8;
	}
	if (i>>28 == 0) {
		n += 4;
		i <<= 4;
	}
	if (i>>30 == 0) {
		n += 2;
		i <<= 2;
	}
	return n - int(i>>31);
}

func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b;
}

func min(a IP, b IP) IP {
	if a < b {
		return a
	}
	return b
}

func max(a IP, b IP) IP {
	if a < b {
		return b
	}
	return a
}

func print_ip(t IP) {
	fmt.Printf("%d.%d.%d.%d", t>>24, t>>16&255, t>>8&255, t&255);
}

func to_ip(a *[4]int) IP {
	return IP(a[0]<<24 | a[1]<<16 | a[2]<<8 | a[3])
}

func print_signle_ip_range(ip IP) {
	print_ip(ip);
	fmt.Print("-");
	print_ip(ip);
	fmt.Println();
}

func print_as_range(p *Range) {
	print_ip(p.start);
	fmt.Print("-");
	print_ip(p.end);
	fmt.Println();
}

func print_signle_cidr(ip IP) {
	print_ip(ip);
	fmt.Println("/32");
}

var printer IpPrinter = print_ip;

func print_as_cidr(p *Range) {
	e := p.end
	cur := p.start
	for {
		var z int;
		var end IP;
		if (cur != 0) {
			z = __builtin_ctz(uint32(cur));
			end = cur + (1 << uint(z)) - 1;
		} else {
			z = 32;
			end = MAX_IP;
		}
		if (end <= e) {
			print_ip(cur);
			fmt.Print("/");
			fmt.Printf("%d\n", 32-z);
		} else {
			for cur <= e {
				if (cur == e) {
					print_signle_cidr(e);
					break;
				}
				z = MinInt(31-__builtin_clz(uint32(e-cur+1)), z);
				end = cur + (1 << uint(z)) - 1;
				print_ip(cur);
				fmt.Print("/");
				fmt.Printf("%d\n", 32-z);
				if (end == MAX_IP) {
					break
				}
				cur = end + 1
				if (cur > e) {
					break
				}
			}
			break;
		}
		if (end == MAX_IP) {
			break
		}
		cur = end + 1
		if (cur > e) {
			break
		}
	}
}

func usage() {
	fmt.Println("usage");
}

func (s Ranges) Len() int      { return len(s) }
func (s Ranges) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (s Ranges) Less(i, j int) bool {
	if (s[i].start < s[j].start) {
		return true
	}
	//    if (s.pairs[i].start > s.pairs[j].start) { return false }
	//    if (s.pairs[i].end < s.pairs[j].end) { return true }
	return false;
}

func main() {
	ot := ot_cidr
	output := print_as_cidr
	var outputFile string
	force := false

	i, length := 1, len(os.Args)
	for ; i < length; i += 1 {
		switch arg := os.Args[i]; arg {
		case "--cidr":
			ot = ot_cidr
			continue
		case "--range":
			ot = ot_range;
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

	switch (ot) {
	case ot_cidr:
		output = print_as_cidr;
		if (force) {
			printer = print_signle_cidr;
		}
	case ot_range:
		output = print_as_range;
		if (force) {
			printer = print_signle_ip_range;
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
	input := bufio.NewScanner(os.Stdin);
	for input.Scan() {
		buf := input.Text()
		var a [4]int
		cnt, err := fmt.Sscanf(buf, "%d.%d.%d.%d", &a[0], &a[1], &a[2], &a[3]);
		if (err != nil) {
			fmt.Fprintf(os.Stderr, "ignore line '%s'\n", buf)
			continue;
		}
		start := to_ip(&a)
		end := start;
		cidr := 32

		cnt, err = fmt.Sscanf(buf, "%d.%d.%d.%d/%d", &a[0], &a[1], &a[2], &a[3], &cidr);
		if cnt == 5 {
			if cidr == 0 {
				start = 0
				end = MAX_IP
			} else {
				start = IP(uint(start) & (0xFFFFFFFF << uint(32-cidr)))
				end = IP(uint(start) + (1 << uint(32-cidr)) - 1)
			}
		} else {
			cnt, err = fmt.Sscanf(buf, "%d.%d.%d.%d-%d.%d.%d.%d", &a[0], &a[1], &a[2], &a[3], &a[0], &a[1], &a[2], &a[3])
			if cnt == 8 {
				end = to_ip(&a);
				if (uint(end) < uint(start)) {
					start, end = end, start
				}
			}
		}
		arr = append(arr, Range{
			start: start,
			end:   end,
		})
	}

	len := len(arr)
	if (len > 0) {
		sort.Sort(Ranges(arr))
		j := 0
		for i := 1; i < len; i += 1 {
			if (arr[i].start <= arr[j].end+1 || arr[j].end == MAX_IP) {
				arr[j].end = max(arr[j].end, arr[i].end);
			} else {
				j += 1
				arr[j] = arr[i]
			}
		}
		len = j + 1;
	}
	for i := 0; i < len; i += 1 {
		output(&arr[i])
	}
}
