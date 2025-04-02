package server

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ev-the-dev/redis-go-clone/resp"
)

type SetOptions struct {
	Expiry  time.Time
	GET     bool
	KEEPTTL bool
	NX      bool
	XX      bool
}

// NOTE: not sure if I should define a new error type like ErrParseSet for all of the errors here
func parseSETOptions(msgs []*resp.Message) (*SetOptions, error) {
	opts := &SetOptions{}

	for i, m := range msgs {
		if m.Type != resp.BulkString && m.Type != resp.SimpleString {
			return nil, fmt.Errorf("%s SET option: parse: expected (%s|%s) but received (%s)", ErrCmdPrefix, resp.BulkString, resp.SimpleString, m.Type)
		}

		opt := strings.ToUpper(m.String)
		switch opt {
		case "EX", "EXAT", "PX", "PXAT":
			if i+1 >= len(msgs) {
				return nil, fmt.Errorf("%s SET option: EX/PX/EXAT/PXAT provided without arg", ErrCmdPrefix)
			}
			if opts.KEEPTTL {
				return nil, fmt.Errorf("%s SET option: KEEPTTL with EX/PX/EXAT/PXAT provided", ErrCmdPrefix)
			}
			if msgs[i+1].Type != resp.BulkString {
				return nil, fmt.Errorf("%s SET option: parse: expected (%s|%s) but received (%s)", ErrCmdPrefix, resp.BulkString, resp.SimpleString, m.Type)
			}
			exp, err := parseSETOptionWithArg(opt, msgs[i+1].String)
			if err != nil {
				return nil, fmt.Errorf("%s SET option: EX/PX/EXAT/PXAT invalid arg: %w", ErrCmdPrefix, err)
			}
			if exp.Before(time.Now()) {
				return nil, fmt.Errorf("%s SET option: expiry set in past", ErrCmdPrefix)
			}
			opts.Expiry = exp
			i++
		case "GET":
			opts.GET = true
		case "KEEPTTL":
			if !opts.Expiry.IsZero() {
				return nil, fmt.Errorf("%s SET option: KEEPTTL with EX/PX/EXAT/PXAT provided", ErrCmdPrefix)
			}
			opts.KEEPTTL = true
		case "NX":
			if opts.XX {
				return nil, fmt.Errorf("%s SET option: NX with XX provided", ErrCmdPrefix)
			}
			opts.NX = true
		case "XX":
			if opts.NX {
				return nil, fmt.Errorf("%s SET option: NX with XX provided", ErrCmdPrefix)
			}
			opts.XX = true
		default:
			return nil, fmt.Errorf("%s SET option: unsupported option: %s", ErrCmdPrefix, opt)
		}
	}

	return opts, nil
}

func parseSETOptionWithArg(opt string, rawArg string) (time.Time, error) {
	arg, err := strconv.ParseInt(rawArg, 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	switch opt {
	case "EX":
		return time.Now().Add(time.Duration(arg) * time.Second), nil
	case "PX":
		return time.Now().Add(time.Duration(arg) * time.Millisecond), nil
	case "EXAT":
		return time.Unix(arg, 0), nil
	case "PXAT":
		return time.UnixMilli(arg), nil
	default:
		return time.Time{}, errors.New("should not have gotten here: unsupported option")
	}
}
