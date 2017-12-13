package pkg

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"io"
)

type fileEntry struct {
	Name   string
	Offset int64
	Size   int64
	Flags  uint32
	Key    cipher.Block
}

func (h *fileEntry) FileType() FileTypeEnum {
	flag := h.Flags & 0xff
	return FileTypeEnum(flag)
}

func (h *fileEntry) KeyType() uint16 {
	flag := h.Flags >> 24 & 0xff
	return uint16(flag)
}

func (h *fileEntry) IsFile() bool {
	switch h.FileType() {
	case FileTypeFile0:
		fallthrough
	case FileTypeFile1:
		fallthrough
	case FileTypeFileEdat:
		fallthrough
	case FileTypeFile3:
		fallthrough
	case FileTypeFileDocinfo:
		fallthrough
	case FileTypeFilePbp:
		fallthrough
	case FileTypeFileModule:
		fallthrough
	case FileTypeFile15:
		fallthrough
	case FileTypeFileKeystone:
		fallthrough
	case FileTypeFilePfs:
		fallthrough
	case FileTypeFileTemp:
		fallthrough
	case FileTypeFileInst:
		fallthrough
	case FileTypeFileClearsign:
		fallthrough
	case FileTypeFileSys:
		fallthrough
	case FileTypeFileDigs:
		return true
	default:
		return false
	}
}

func (h *fileEntry) IsDirectory() bool {
	switch h.FileType() {
	case FileTypeDirectory:
		fallthrough
	case FileTypeDirectoryPfs:
		return true
	default:
		return false
	}
}

func (pr *Reader) readFileIndex() ([]fileEntry, error) {
	itemRecords := make([]ItemRecord, pr.FileHeader.ItemCount)
	err := binary.Read(pr.reader, binary.BigEndian, &itemRecords)
	if err != nil {
		return nil, err
	}

	recordListSize := binary.Size(itemRecords)
	tableSize := itemRecords[0].DataOffset - int64(itemRecords[0].FilenameOffset)
	tableBuffer := make([]byte, tableSize)
	// do not use the current aes reader since the names
	// table could be encrypted using different keys
	_, err = io.ReadFull(pr.aesReader.RawReader(), tableBuffer)
	if err != nil {
		return nil, err
	}

	// advance the read stream
	pr.aesReader.SetCounter((int64(recordListSize) + tableSize) / 16)

	var ps3ctr cipher.Block

	if pr.extendedHeader.KeyType() == 1 {
		ps3ctr, err = aes.NewCipher(KeyPS3)
		if err != nil {
			return nil, err
		}
	}

	entries := make([]fileEntry, pr.FileHeader.ItemCount)

	for idx, entry := range itemRecords {
		counter := int64(entry.FilenameOffset / 16)
		tableOffset := int(entry.FilenameOffset) - recordListSize
		encryptedName := tableBuffer[tableOffset : tableOffset+int(entry.FilenameSize)]

		var ctr cipher.Block
		if pr.pkgType == PackageTypePSP || pr.pkgType == PackageTypePSOne {
			if entry.KeyType() == EntryTypePSP {
				ctr = pr.aesReader.block
			} else {
				ctr = ps3ctr
			}
		} else {
			ctr = pr.aesReader.block
		}

		AESCTRDecrypt(ctr, encryptedName, encryptedName, pr.FileHeader.DataIV[:], counter)

		entries[idx].Name = string(encryptedName)
		entries[idx].Size = entry.DataSize
		entries[idx].Offset = entry.DataOffset
		entries[idx].Flags = entry.Flags
		entries[idx].Key = ctr
	}

	return entries, nil
}
