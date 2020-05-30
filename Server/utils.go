package main

import "strings"

type parseFlag int

const (
	ok                 parseFlag = 0
	invalidParamsCount parseFlag = 1
	notMD              parseFlag = 2
)

//parseUtilsAndSignal parses the given string in line with the MD protocol. if expectedLen is 0 it is disregarded and any number of fields more than 0 are accepted.
func parseUtilsAndSignal(text string, expectedLen int) ([]string, parseFlag) {
	fields := strings.Fields(text)
	flen := len(fields)
	if flen == 0 {
		return nil, invalidParamsCount
	}
	if expectedLen != 0 {
		if flen != expectedLen {
			return nil, invalidParamsCount
		}
	}
	if fields[0] != "MD" {
		return nil, notMD
	}

	return fields, ok
}
