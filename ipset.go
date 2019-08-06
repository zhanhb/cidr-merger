package main

import (
	"bytes"
	"math/bits"
	"net"
	"sort"
)

type IRange interface {
	ToIp() net.IP // return nil if can't be represented as a single ip
	ToIpNets() []*net.IPNet
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
func (r *Range) ToIpNets() []*net.IPNet {
	s, end := r.start, r.end
	ipBits := len(s) * 8
	if allZero(s) && allFF(end) {
		return []*net.IPNet{
			{IP: s, Mask: net.CIDRMask(0, ipBits)},
		}
	}
	var result []*net.IPNet
	for {
		// assert s <= end;
		// will never overflow
		cidr := max(leadingZero(addOne(minus(end, s)))+1, ipBits-trailingZeros(s))
		ipNet := &net.IPNet{IP: s, Mask: net.CIDRMask(cidr, ipBits)}
		result = append(result, ipNet)
		tmp := lastIp(ipNet)
		if !lessThan(tmp, end) {
			return result
		}
		s = addOne(tmp)
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

func (r IpWrapper) ToIp() net.IP {
	return r.IP
}
func (r IpWrapper) ToIpNets() []*net.IPNet {
	ipBits := len(r.IP) * 8
	return []*net.IPNet{
		{IP: r.IP, Mask: net.CIDRMask(ipBits, ipBits)},
	}
}
func (r IpWrapper) ToRange() *Range {
	return &Range{start: r.IP, end: r.IP}
}

type IpNetWrapper struct {
	*net.IPNet
}

func (r IpNetWrapper) ToIp() net.IP {
	if ones, bts := r.IPNet.Mask.Size(); ones == bts {
		return r.IPNet.IP
	}
	return nil
}
func (r IpNetWrapper) ToIpNets() []*net.IPNet {
	return []*net.IPNet{r.IPNet}
}
func (r IpNetWrapper) ToRange() *Range {
	ipNet := r.IPNet
	return &Range{start: ipNet.IP, end: lastIp(ipNet)}
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
		panic("assert failed: unexpected IPNet " + ipNet.String())
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
		panic("assert failed: unexpected ip " + ip.String())
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
		panic("assert failed: subtract " + b.String() + " from " + a.String())
	}
	return result
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

type Ranges []*Range

func (s Ranges) Len() int { return len(s) }
func (s Ranges) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s Ranges) Less(i, j int) bool {
	return lessThan(s[i].start, s[j].start)
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

func singleOrSelf(r IRange) IRange {
	if ip := r.ToIp(); ip != nil {
		return IpWrapper{ip}
	}
	return r
}

func returnSelf(r IRange) IRange {
	return r
}
