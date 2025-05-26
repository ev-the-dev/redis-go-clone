package rdb

type ErrPrefix string

const (
	ErrLoadPrefix         ErrPrefix = "rdb: load:"
	ErrLengthEncodePrefix ErrPrefix = "rdb: parse: length encoding:"
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
)
