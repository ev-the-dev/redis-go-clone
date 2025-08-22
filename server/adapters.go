package server

import (
	"fmt"
	"strings"
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
		// TODO: need to parse the `m.Array` into an array of `store.Record`s
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

func toRESPString(r *store.Record) (string, error) {
	var b strings.Builder
	switch r.Type {
	case resp.Array, resp.Sets:
		for _, v := range r.Value.([]*store.Record) {
			nestedValue, err := toRESPString(v)
			if err != nil {
				return "", fmt.Errorf("%s unable to adapt nested array: %+v", ErrAdaptPrefix, v)
			}
			b.WriteString(nestedValue)
		}
	case resp.Booleans:
		b.WriteString(resp.EncodeBoolean(r.Value.(bool)))
	case resp.BulkString:
		b.WriteString(resp.EncodeBulkString(r.Value.(string)))
	case resp.SimpleString:
		b.WriteString(resp.EncodeSimpleString(r.Value.(string)))
	case resp.Integer:
		b.WriteString(resp.EncodeInteger(r.Value.(int)))
	case resp.Maps:
		v = m.Map
	case resp.Nulls:
		v = nil
	default:
		return "", fmt.Errorf("%s unsupported type (%s) from store record: %+v", ErrAdaptPrefix, r.Type.String(), r)
	}

}
