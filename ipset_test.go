package main

import "testing"

func TestAddOne(t *testing.T) {
	data := [][]string{
		{"0.0.0.0", "0.0.0.1"},
		{"1.255.254.255", "1.255.255.0"},
		{"1.255.255.255", "2.0.0.0"},
		{"255.255.255.255", "0.0.0.0"},
		{"::", "::1"},
		{"::ffff", "::1:0"},
		{"ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff", "::"},
	}
	for _, item := range data {
		in := parseIp(item[0])
		expect := parseIp(item[1])
		result := addOne(in)
		assertEqualsF(expect, result, "expect %s + 1 => %s, but got %s", in, expect, result)
	}
}

func TestIpNetWrapper_String(t *testing.T) {
	inputs := []string{
		"0.0.0.0/0",
		"0.0.0.0/1",
		"0.0.0.0/32",
		"128.0.0.0/1",
		"0.0.1.0/24",
		"192.0.1.0/24",
		"::/0",
	}
	for _, text := range inputs {
		iRange, err := parse(text)
		assert(err == nil, "err is %v", err)
		ipNetWrapper := iRange.(IpNetWrapper)
		assertEquals(text, ipNetWrapper.String())
	}
}
