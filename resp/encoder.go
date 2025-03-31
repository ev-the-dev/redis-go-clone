package resp

import "fmt"

// NOTE: for all encoders I might need to implement
// some way to validate or strip RESP data signifiers.
// The encoders assume Go data types as args.
// i.e.
// valid: "hello"
// invalid: "+hello" -- unless '+' is meant as literal part of message.
//
// Although, perhaps that shouldn't be the concern of these functions.

func EncodeBulkString(s string) string {
	return fmt.Sprintf("$%d\r\n%s\r\n", len(s), s)
}

// RESP2 Specific Type
func EncodeNullBulkString() string {
	return fmt.Sprint("$-1\r\n")
}

func EncodeSimpleErr(s string) string {
	return fmt.Sprintf("-ERR %s\r\n", s)
}

func EncodeSimpleString(s string) string {
	return fmt.Sprintf("+%s\r\n", s)
}
