package server

import (
	"fmt"
	"time"

	"github.com/ev-the-dev/redis-go-clone/rdb"
	"github.com/ev-the-dev/redis-go-clone/resp"
	"github.com/ev-the-dev/redis-go-clone/store"
)

func fromRDB(e *rdb.Entry) (*store.Record, error) {
	var rT resp.RESPType

	switch e.ValType {
	case rdb.StringEncoded:
		rT = resp.BulkString
	case rdb.ListEncoded:
		rT = resp.Array
	case rdb.SetEncoded, rdb.SortedSetEncoded:
		rT = resp.Sets
	case rdb.HashEncoded:
		rT = resp.Maps
	default:
		return nil, fmt.Errorf("%s unsupported rdb type (%s) for entry: %+v", ErrAdaptPrefix, e.ValType.String(), e)
	}

	// NOTE: If need be we could also return the Key here
	return &store.Record{
		ExpiresAt: e.Expire,
		Type:      rT,
		Value:     e.Val,
	}, nil
}

func fromRESP(m *resp.Message, expiry time.Time) (*store.Record, error) {
	var v any

	switch m.Type {
	case resp.Array, resp.Sets:
		v = m.Array
	case resp.Booleans:
		v = m.Boolean
	case resp.BulkString, resp.SimpleString:
		v = m.String
	case resp.Integer:
		v = m.Integer
	case resp.Maps:
		v = m.Map
	case resp.Nulls:
		v = nil
	default:
		return nil, fmt.Errorf("%s unsupported resp type (%s) for message: %+v", ErrAdaptPrefix, m.Type.String(), m)
	}

	return &store.Record{
		ExpiresAt: expiry,
		Type:      m.Type,
		Value:     v,
	}, nil
}
