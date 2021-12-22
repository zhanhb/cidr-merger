package main

import "fmt"

func assert(condition bool, format string, args ...interface{}) {
	if !condition {
		panic(fmt.Sprintf("assert failed: "+format, args...))
	}
}
