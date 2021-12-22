package main

import (
	"reflect"
	"testing"
)

func assertEqualsF(expect, result interface{}, format string, args ...interface{}) {
	assert(reflect.DeepEqual(expect, result), format, args...)
}

func assertEquals(expect, result interface{}) {
	assertEqualsF(expect, result, "expect %v, but got %v", expect, result)
}

func getAssertPanic() (res interface{}) {
	defer func() { res = recover() }()
	assert(false, "%v%v%v", 1, 2, 3)
	return
}

func TestAssertEquals(t *testing.T) {
	assertEquals("assert failed: 123", getAssertPanic())
}
