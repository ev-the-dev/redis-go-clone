package rdb

type ErrPrefix string

const (
	ErrLoadPrefix                ErrPrefix = "rdb: load:"
	ErrLengthEncodePrefix        ErrPrefix = "rdb: parse: length encoding:"
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
