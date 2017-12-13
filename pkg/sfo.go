package pkg

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"strconv"
)

var sfoMagic = [4]byte{0x00, 0x50, 0x53, 0x46}

const (
	utf8Special uint16 = 0x0004
	utf8        uint16 = 0x0204
	integer     uint16 = 0x0404
)

type sfoHeader struct {
	Magic             [4]byte
	Version           int32
	KeyTableOffset    int32
	DataTableOffset   int32
	IndexTableEntries int32
}

type sfoIndexTableEntry struct {
	KeyOffset      uint16
	ParamFormat    uint16
	ParamLength    uint32
	ParamMaxLength uint32
	DataOffset     uint32
}

func (pr *Reader) readSFO(r io.Reader) (n int64, err error) {
	var header sfoHeader
	err = binary.Read(r, binary.LittleEndian, &header)
	if err != nil {
		return
	}

	n = int64(binary.Size(header))

	if !bytes.Equal(header.Magic[:], sfoMagic[:]) {
		err = errors.New("invalid SFO header")
		return
	}

	index := make([]sfoIndexTableEntry, header.IndexTableEntries)
	err = binary.Read(r, binary.LittleEndian, &index)
	if err != nil {
		return
	}

	n += int64(binary.Size(index))

	keys := make([]byte, header.DataTableOffset-header.KeyTableOffset)
	_, err = io.ReadFull(r, keys)
	if err != nil {
		return
	}

	n += int64(binary.Size(keys))

	last := index[len(index)-1]
	valuesSize := last.DataOffset + last.ParamMaxLength

	values := make([]byte, valuesSize)
	_, err = io.ReadFull(r, values)
	if err != nil {
		return
	}

	n += int64(valuesSize)

	pr.SfoEntries = map[string]string{}

	for _, entry := range index {
		switch entry.ParamFormat {
		case utf8Special:
			n := bytes.IndexByte(keys[entry.KeyOffset:], 0)
			key := string(keys[entry.KeyOffset : int(entry.KeyOffset)+n])
			value := string(values[entry.DataOffset : entry.DataOffset+entry.ParamLength])
			pr.SfoEntries[key] = value
		case utf8:
			n := bytes.IndexByte(keys[entry.KeyOffset:], 0)
			key := string(keys[entry.KeyOffset : int(entry.KeyOffset)+n])
			value := string(values[entry.DataOffset : entry.DataOffset+entry.ParamLength-1])
			pr.SfoEntries[key] = value
		case integer:
			n := bytes.IndexByte(keys[entry.KeyOffset:], 0)
			key := string(keys[entry.KeyOffset : int(entry.KeyOffset)+n])
			value := binary.LittleEndian.Uint32(values[entry.DataOffset : entry.DataOffset+entry.ParamLength])
			pr.SfoEntries[key] = strconv.Itoa(int(value))
		}
	}

	return
}
