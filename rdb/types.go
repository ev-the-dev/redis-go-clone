package rdb

import "time"

type Entry struct {
	Expire  time.Time
	Key     any
	KeyType ValueType
	Val     any
	ValType ValueType
}

type ErrPrefix string

const (
	ErrLoadPrefix                ErrPrefix = "rdb: load:"
	ErrLengthEncodePrefix        ErrPrefix = "rdb: parse: length encoding:"
	ErrParseDataPrefix           ErrPrefix = "rdb: parse: data:"
	ErrReadHeader                ErrPrefix = "rdb: read: header:"
	ErrReadMetadata              ErrPrefix = "rdb: read: metadata:"
	ErrReadDatabase              ErrPrefix = "rdb: read: database:"
	ErrReadFooter                ErrPrefix = "rdb: read: footer:"
	ErrSpecialLengthEncodePrefix ErrPrefix = "rdb: parse: length encoding: special format:"
)

type SpecialLengthType byte

const (
	SpecialInt8 SpecialLengthType = iota
	SpecialInt16
	SpecialInt32
	SpecialLZF
)

type ValueType byte

const (
	StringEncoded ValueType = iota
	ListEncoded
	SetEncoded
	SortedSetEncoded
	HashEncoded
	_
	_
	_
	_
	ZipmapEncoded
	ZiplistEncoded
	IntsetEncoded
	ZiplistSortedSetEncoded
	ZiplistHashmapEncoded
	QuicklistListEncoded
	_
	ErrEncoded
)
