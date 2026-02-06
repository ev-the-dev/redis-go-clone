package resp

import (
	"fmt"
	"strings"
)

// NOTE: for all encoders I might need to implement
// some way to validate or strip RESP data signifiers.
// The encoders assume Go data types as args.
// i.e.
// valid: "hello"
// invalid: "+hello" -- unless '+' is meant as literal part of message.
//
// Although, perhaps that shouldn't be the concern of these functions.

func EncodeArray(length int, ss ...string) string {
	s := strings.Join(ss, "")
	return fmt.Sprintf("*%d\r\n%s", length, s)
}

func EncodeBoolean(b bool) string {
	bS := ""
	if b {
		bS = "t"
	} else {
		bS = "f"
	}
	return fmt.Sprintf("#%s\r\n", bS)
}

func EncodeBulkString(s string) string {
	return fmt.Sprintf("$%d\r\n%s\r\n", len(s), s)
}

func EncodeInteger(n int) string {
	return fmt.Sprintf(":%d\r\n", n)
}

func EncodeMap(length int, m string) string {
	return fmt.Sprintf("%%%d\r\n%s", length, m)
}

func EncodeNullArray() string {
	return fmt.Sprintf("*-1\r\n")
}

// RESP2 Specific Type
func EncodeNullBulkString() string {
	return fmt.Sprint("$-1\r\n")
}

func EncodeNulls() string {
	return fmt.Sprintf("_\r\n")
}

func EncodeSimpleErr(s string) string {
	return fmt.Sprintf("-ERR %s\r\n", s)
}

func EncodeSimpleString(s string) string {
	return fmt.Sprintf("+%s\r\n", s)
}
