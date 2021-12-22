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
