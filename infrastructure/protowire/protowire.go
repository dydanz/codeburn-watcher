package protowire

import (
	"encoding/binary"
	"fmt"
)

// readVarint decodes a protobuf varint from buf at offset pos.
// Returns the value and the new offset.
func ReadVarint(buf []byte, pos int) (uint64, int, error) {
	val, n := binary.Uvarint(buf[pos:])
	if n <= 0 {
		return 0, pos, fmt.Errorf("protowire: invalid varint at %d", pos)
	}
	return val, pos + n, nil
}

// Field wire types
const (
	WireVarint  = 0
	WireFixed64 = 1
	Wire2       = 2 // length-delimited
	WireFixed32 = 5
)

// Field represents one decoded protobuf field.
type Field struct {
	Number uint32
	Type   int
	Data   []byte  // for Wire2
	Int    uint64  // for WireVarint
}

// DecodeMessage decodes a flat (non-nested) protobuf message.
// Returns all top-level fields.
func DecodeMessage(buf []byte) ([]Field, error) {
	var fields []Field
	pos := 0
	for pos < len(buf) {
		tag, newPos, err := ReadVarint(buf, pos)
		if err != nil {
			return fields, nil // treat as end-of-message
		}
		pos = newPos
		fieldNum := uint32(tag >> 3)
		wireType := int(tag & 0x7)

		switch wireType {
		case WireVarint:
			v, newPos, err := ReadVarint(buf, pos)
			if err != nil {
				return fields, nil
			}
			fields = append(fields, Field{Number: fieldNum, Type: wireType, Int: v})
			pos = newPos
		case WireFixed64:
			if pos+8 > len(buf) {
				return fields, nil
			}
			fields = append(fields, Field{Number: fieldNum, Type: wireType, Int: binary.LittleEndian.Uint64(buf[pos : pos+8])})
			pos += 8
		case Wire2:
			l, newPos, err := ReadVarint(buf, pos)
			if err != nil {
				return fields, nil
			}
			pos = newPos
			if int(l) > len(buf)-pos {
				return fields, nil
			}
			data := make([]byte, l)
			copy(data, buf[pos:pos+int(l)])
			fields = append(fields, Field{Number: fieldNum, Type: wireType, Data: data})
			pos += int(l)
		case WireFixed32:
			if pos+4 > len(buf) {
				return fields, nil
			}
			pos += 4
		default:
			return fields, nil
		}
	}
	return fields, nil
}

// IntField finds the first int value for the given field number.
func IntField(fields []Field, num uint32) (uint64, bool) {
	for _, f := range fields {
		if f.Number == num && f.Type == WireVarint {
			return f.Int, true
		}
	}
	return 0, false
}

// StrField finds the first string value for the given field number.
func StrField(fields []Field, num uint32) (string, bool) {
	for _, f := range fields {
		if f.Number == num && f.Type == Wire2 {
			return string(f.Data), true
		}
	}
	return "", false
}

// MsgField finds the first nested message bytes for the given field number.
func MsgField(fields []Field, num uint32) ([]byte, bool) {
	for _, f := range fields {
		if f.Number == num && f.Type == Wire2 {
			return f.Data, true
		}
	}
	return nil, false
}
