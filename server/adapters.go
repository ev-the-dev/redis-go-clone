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
	sR := &store.Record{
		ExpiresAt: e.Expire,
	}

	switch e.ValType {
	case rdb.StringEncoded:
		sR.Type = store.StringType
		sR.String = e.Val.(string)
	case rdb.ListEncoded:
		sR.Type = store.ArrayType
		// TODO: I believe I need to recurse over `fromRDB` to
		// appropriately extract all the rdb's nested data.
		sR.Array = e.Val.([]any)
	case rdb.SetEncoded, rdb.SortedSetEncoded:
		sR.Type = store.SetType
		sR.Array = e.Val.([]any)
	case rdb.HashEncoded:
		sR.Type = store.MapType
		sR.Map = e.Val.(map[string]any)
	default:
		return nil, fmt.Errorf("%s unsupported rdb type (%s) for entry: %+v", ErrAdaptPrefix, e.ValType.String(), e)
	}

	return sR, nil
}

// TODO: Think about removing `expiry` from this as there are tons of cases
// where we'd use this but the Message type doesn't warrant an expiration.
// Instead opt for a method on `*store.Record#WithExpiry`.
func fromRESP(m *resp.Message, expiry time.Time) (*store.Record, error) {
	sR := &store.Record{
		ExpiresAt: expiry,
	}

	switch m.Type {
	// TODO: Have to revisit how resp Sets behave and adapt accordingly
	case resp.Array:
		rS, err := fromRESPArrayToStoreArray(m, expiry)
		if err != nil {
			return nil, fmt.Errorf("%s from resp: case array: %w", ErrAdaptPrefix, err)
		}
		sR.Type = store.ArrayType
		sR.Array = rS
	case resp.Booleans:
		sR.Type = store.BooleanType
		sR.Boolean = m.Boolean
	case resp.BulkString, resp.SimpleString:
		sR.Type = store.StringType
		sR.String = m.String
	case resp.Integer:
		sR.Type = store.IntegerType
		sR.Integer = m.Integer
	case resp.Maps:
		sM, err := fromRESPMapToStoreMap(m, expiry)
		if err != nil {
			return nil, fmt.Errorf("%s from resp: case map: %w", ErrAdaptPrefix, err)
		}
		sR.Type = store.MapType
		sR.Map = sM
	case resp.Nulls:
		sR.Type = store.NilType
	case resp.Sets:
		rS, err := fromRESPArrayToStoreArray(m, expiry)
		if err != nil {
			return nil, fmt.Errorf("%s from resp: case set: %w", ErrAdaptPrefix, err)
		}
		sR.Type = store.SetType
		sR.Array = rS
	default:
		return nil, fmt.Errorf("%s unsupported resp type (%s) for message: %+v", ErrAdaptPrefix, m.Type.String(), m)
	}

	return sR, nil
}

func fromRESPArrayToStoreArray(m *resp.Message, expiry time.Time) ([]*store.Record, error) {
	if m.Type != resp.Array && m.Type != resp.Sets {
		return nil, fmt.Errorf("%s trying to adapt from RESP (Array|Set) but got (%s)", ErrAdaptPrefix, m.Type.String())
	}

	rS := make([]*store.Record, len(m.Array))
	for _, v := range m.Array {
		sR, err := fromRESP(v, expiry)
		if err != nil {
			return nil, fmt.Errorf("%s from resp: array: %w", ErrAdaptPrefix, err)
		}

		rS = append(rS, sR)
	}

	return rS, nil
}

func fromRESPMapToStoreMap(m *resp.Message, expiry time.Time) (map[string]*store.Record, error) {
	if m.Type != resp.Maps {
		return nil, fmt.Errorf("%s trying to adapt from RESP (Map) but got (%s)", ErrAdaptPrefix, m.Type.String())
	}

	sM := make(map[string]*store.Record)
	for k, v := range m.Map {
		storeValue, err := fromRESP(v, expiry)
		if err != nil {
			return nil, fmt.Errorf("%s from resp: map value: %w", ErrAdaptPrefix, err)
		}

		sM[k] = storeValue
	}

	return sM, nil
}

// NOTE: Go doesn't allow negative indices for array/slice accessing.
// This adapter helps support Redis functionality that does allow it.
func NormalizeIndex(idx int, length int) int {
	if idx >= 0 {
		return idx
	}

	// NOTE: Just incase a negative index that is larger than
	// the length is passed in, we just want to anchor it at
	// 0 (zero) instead.
	idx = max(length+idx, 0)

	return idx
}

// NOTE: Might have this simply output a single string that is already
// encoded as an Array. I'm not sure if this function will be used for
// any other scenario.
//
// Currently it outputs a slice of RESP strings, but not in itself
// a RESP string, so the output of this function still needs to be
// encoded before reaching the client.
func toBulkRESPString(r []*store.Record) ([]string, error) {
	ss := make([]string, len(r))
	for i, v := range r {
		s, err := toRESPString(v)
		if err != nil {
			return nil, fmt.Errorf("%s bulk to resp string: %w", ErrAdaptPrefix, err)
		}

		ss[i] = s
	}

	return ss, nil
}

func toRESPString(r *store.Record) (string, error) {
	var b strings.Builder
	switch r.Type {
	case store.ArrayType, store.SetType:
		arrVal := r.Array
		for _, v := range arrVal {
			nestedValue, err := toRESPString(v)
			if err != nil {
				return "", fmt.Errorf("%s unable to adapt nested array: %+v", ErrAdaptPrefix, v)
			}
			b.WriteString(nestedValue)
		}
		return resp.EncodeArray(len(arrVal), b.String()), nil
	case store.BooleanType:
		b.WriteString(resp.EncodeBoolean(r.Boolean))
	case store.StringType:
		b.WriteString(resp.EncodeBulkString(r.String))
	case store.IntegerType:
		b.WriteString(resp.EncodeInteger(r.Integer))
	case store.MapType:
		s, err := toRESPStringFromStoreMap(b, r.Map)
		if err != nil {
			return "", fmt.Errorf("%s to resp: %w", ErrAdaptPrefix, err)
		}
		b.WriteString(s)
	case store.NilType:
		b.WriteString(resp.EncodeNulls())
	default:
		return "", fmt.Errorf("%s unsupported type (%s) from store record: %+v", ErrAdaptPrefix, r.Type.String(), r)
	}

	return b.String(), nil
}

func toRESPStringFromStoreMap(b strings.Builder, m map[string]*store.Record) (string, error) {
	for k, v := range m {
		// Keys are already strings, encode them as BulkStrings
		b.WriteString(resp.EncodeBulkString(k))

		nestedValue, err := toRESPString(v)
		if err != nil {
			return "", fmt.Errorf("%s unable to adapt map value: %+v", ErrAdaptPrefix, v)
		}
		b.WriteString(nestedValue)
	}
	return resp.EncodeMap(len(m), b.String()), nil
}
