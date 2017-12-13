package pkg

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"os"
)

type PackageType int

const (
	PackageTypePSOne     PackageType = 1 << iota
	PackageTypePSP
	PackageTypeVitaApp
	PackageTypeVitaDLC
	PackageTypeVitaPatch
	PackageTypePSM
)

var fileHeader = [4]byte{0x7F, 0x50, 0x4B, 0x47}
var extHeader = [4]byte{0x7F, 0x65, 0x78, 0x74}

type indexData struct {
	itemRecords []fileEntry
	idx         int
}

type Reader struct {
	// pkg reader
	reader io.Reader
	// current file reader inside the pkg
	current numBytesReader
	pad     int64
	err     error
	// file table
	index   indexData
	pkgType PackageType

	// raw pkg reader
	rawReader io.Reader
	// decrypted pkg reader
	aesReader *ctrStream
	// holds the calculated sha1sum
	hasher hash.Hash
	// pkg header
	FileHeader     FileHeader
	extendedHeader ExtendedHeader
	meta           Metadata
	SfoEntries     map[string]string
	headBuffer     bytes.Buffer
	tailBuffer     bytes.Buffer
	rif            []byte

	// hashes, calculated and from file
	FileHash       []byte
	CalculatedHash []byte
}

type ReadCloser struct {
	f *os.File
	Reader
}

func OpenReader(name string, rif string) (*ReadCloser, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}

	r := new(ReadCloser)
	if err := r.init(f, rif); err != nil {
		f.Close()
		return nil, err
	}

	r.f = f
	return r, nil
}

func NewReader(r io.Reader, rif string) (*Reader, error) {
	zr := new(Reader)
	if err := zr.init(r, rif); err != nil {
		return nil, err
	}

	return zr, nil
}

func (rc *ReadCloser) Close() error {
	return rc.f.Close()
}

func (pr *Reader) Valid() bool {
	return bytes.Compare(pr.CalculatedHash, pr.FileHash) == 0
}

func (pr *Reader) seekAhead(cur, offset int64) (pos int64, err error) {
	count := offset - cur

	if count < 0 {
		err = errors.New("the given offset is behind the current position")
		return
	} else if count == 0 {
		pos = cur + count
		return
	}

	_, err = io.CopyN(ioutil.Discard, pr.reader, count)
	if err != nil {
		return
	}

	pos = cur + count

	return
}

func (pr *Reader) PackageType() PackageType {
	return pr.pkgType
}

func (pr *Reader) readMetadata(cur int64) (pos int64, err error) {
	pos, err = pr.seekAhead(cur, int64(pr.FileHeader.InfoOffset))
	if err != nil {
		return
	}

	for i := 0; i < int(pr.FileHeader.InfoCount); i++ {
		var info InfoHeader

		err = binary.Read(pr.reader, binary.BigEndian, &info)
		if err != nil {
			return
		}

		pos += int64(binary.Size(info))

		var buf []uint32

		if info.Size > 0 {
			buf = make([]uint32, info.Size/4)
			err = binary.Read(pr.reader, binary.BigEndian, &buf)
			if err != nil {
				return
			}

			pos += int64(info.Size)
		}

		switch info.Type {
		case IdentifierDRMType:
			pr.meta.DrmType = buf[0]
		case IdentifierContentType:
			pr.meta.ContentType = ContentTypeEnum(buf[0])
		case IdentifierPackageFlags:
			pr.meta.PackageFlags = buf[0]
		case IdentifierFileIndexInfo:
			pr.meta.IndexTableOffset = buf[0]
			pr.meta.IndexTableSize = buf[1]
		case IdentifierSFO:
			pr.meta.SfoOffset = buf[0]
			pr.meta.SfoSize = buf[1]
		}
	}

	return
}

func (pr *Reader) setupDecryption() error {
	var pkgType PackageType

	switch pr.meta.ContentType {
	case ContentTypePS1:
		pkgType = PackageTypePSOne
	case ContentTypePSP:
		fallthrough
	case ContentTypePSPGo:
		fallthrough
	case ContentTypeMinis:
		fallthrough
	case ContentTypeNeoGeo:
		pkgType = PackageTypePSP
	case ContentTypeVitaApp:
		pkgType = PackageTypeVitaApp
	case ContentTypeVitaDLC:
		pkgType = PackageTypeVitaDLC
	case ContentTypePSM1:
	case ContentTypePSM2:
		pkgType = PackageTypePSM
	default:
		return fmt.Errorf("unsupported package type: %v", pkgType)
	}

	if pkgType == PackageTypeVitaApp && pr.SfoEntries["CATEGORY"] == "gp" {
		pkgType = PackageTypeVitaPatch
	}

	pr.pkgType = pkgType

	var baseKey []byte
	ctrKey := make([]byte, 16)

	switch pr.extendedHeader.KeyType() {
	case 1:
		ctrKey = KeyPSP
	case 2:
		baseKey = VitaKey2
	case 3:
		baseKey = KeyVita3
	case 4:
		baseKey = KeyVita4
	default:
		return fmt.Errorf("unknown key type: %v", pr.extendedHeader.KeyType())
	}

	if pr.extendedHeader.KeyType() != 1 {
		// encrypt the iv
		err := AESECBEncrypt(ctrKey, pr.FileHeader.DataIV[:], baseKey)
		if err != nil {
			return err
		}
	}

	reader, err := NewCTRReader(pr.reader, ctrKey, pr.FileHeader.DataIV[:], 0)
	if err != nil {
		return err
	}

	pr.aesReader = reader

	// reader = raw + hash + head + aes
	pr.reader = reader

	return nil
}

func (pr *Reader) init(r io.Reader, rif string) error {
	pr.hasher = sha1.New()
	// combine the file reader with the hash calculator
	hashReader := io.TeeReader(r, pr.hasher)
	// combine the file reader (and hash calculator) with the head.bin buffer
	headHashReader := io.TeeReader(hashReader, &pr.headBuffer)

	// reader = raw + hash + head
	pr.reader = headHashReader

	// read the pkg header
	err := binary.Read(pr.reader, binary.BigEndian, &pr.FileHeader)
	if err != nil {
		return err
	}

	cur := int64(binary.Size(pr.FileHeader))

	if !bytes.Equal(pr.FileHeader.Magic[:], fileHeader[:]) {
		return errors.New("invalid PKG file")
	}

	// check if the header size can hold both pkg headers
	if pr.FileHeader.HeaderSize <= int32(binary.Size(pr.FileHeader)) {
		return errors.New("unsupported PKG type (no extended header)")
	}

	if pr.FileHeader.ItemCount == 0 {
		return errors.New("PKG has no item entries")
	}

	// read the extender header
	err = binary.Read(pr.reader, binary.BigEndian, &pr.extendedHeader)
	if err != nil {
		return err
	}

	cur += int64(binary.Size(pr.extendedHeader))

	if !bytes.Equal(pr.extendedHeader.Magic[:], extHeader[:]) {
		return errors.New("invalid PKG extended header")
	}

	pr.rawReader = r

	cur, err = pr.readMetadata(cur)
	if err != nil {
		return err
	}

	if pr.meta.SfoOffset > 0 && pr.meta.SfoSize > 0 {
		cur, _ = pr.seekAhead(cur, int64(pr.meta.SfoOffset))
		n, err := pr.readSFO(pr.reader)
		if err != nil {
			return err
		}

		cur += n
	}

	// advance to the first encrypted block
	cur, _ = pr.seekAhead(cur, pr.FileHeader.DataOffset)

	err = pr.setupDecryption()
	if err != nil {
		return err
	}

	if len(rif) > 0 && pr.PackageType() != PackageTypePSOne && pr.PackageType() != PackageTypePSP {
		pr.rif, err = DecodeLicense(rif, pr.PackageType())
		if err != nil {
			return err
		}

		rifid := pr.rifContentID()
		cid := pr.FileHeader.GetContentID()

		if rifid != cid {
			return fmt.Errorf("zRIF content ID '%s' doesn't match pkg '%s'", rifid, cid)
		}
	}

	entries, err := pr.readFileIndex()
	if err != nil {
		return err
	}

	pr.index = indexData{itemRecords: entries, idx: 0}

	// drop the head.bin reader from the chain
	pr.aesReader.SetRawReader(hashReader)

	return nil
}

func (pr *Reader) readTail() error {
	pr.reader = pr.aesReader.RawReader()
	// combine the file reader (and hash calculator) with the head.bin buffer
	tailHashReader := io.TeeReader(pr.reader, &pr.tailBuffer)
	tailOffset := pr.FileHeader.DataOffset + pr.FileHeader.DataSize
	tailSize := pr.FileHeader.TotalSize - tailOffset

	_, err := io.CopyN(ioutil.Discard, tailHashReader, tailSize-0x20)
	if err != nil {
		return err
	}

	pr.CalculatedHash = pr.hasher.Sum(nil)

	tailHashReader = io.TeeReader(pr.rawReader, &pr.tailBuffer)
	fileHash := make([]byte, 0x20)

	_, err = io.ReadFull(tailHashReader, fileHash)
	if err != nil {
		return err
	}

	pr.FileHash = fileHash[0:20]

	return nil
}

func (pr *Reader) HeadWriter(w io.Writer) (int64, error) {
	return pr.headBuffer.WriteTo(w)
}

func (pr *Reader) TailWriter(w io.Writer) (int64, error) {
	return pr.tailBuffer.WriteTo(w)
}

func (pr *Reader) rifContentID() string {
	var offset int
	if pr.PackageType() == PackageTypePSM {
		offset = 0x50
	} else {
		offset = 0x10
	}

	return string(pr.rif[offset: offset+36])
}

func (pr *Reader) GetTitle() string {
	title, exists := pr.SfoEntries["TITLE"]
	if !exists {
		title = pr.SfoEntries["STITLE"]
	}

	return title
}

func (pr *Reader) GetTitleID() string {
	return pr.FileHeader.GetTitleID()
}

func (pr *Reader) GetRegion() string {
	if pr.extendedHeader.KeyType() != 1 {
		// Vita codes, 4th letter of TITLE_ID
		id := pr.FileHeader.ContentID[10]

		if id == 'A' || id == 'E' {
			return "USA"
		}

		if id == 'B' || id == 'F' {
			return "EUR"
		}

		if id == 'C' || id == 'G' {
			return "JPN"
		}

		if id == 'D' || id == 'H' {
			return "ASIA"
		}
	} else {
		// PSP codes, 3rd letter of TITLE_ID
		id := pr.FileHeader.ContentID[9]
		if id == 'U' {
			return "USA"
		}

		if id == 'E' {
			return "EUR"
		}

		if id == 'J' {
			return "JPN"
		}

		if id == 'A' || id == 'H' {
			return "ASIA"
		}
	}

	return "UNK"
}
