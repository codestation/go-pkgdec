package pkg

type FileHeader struct {
	Magic                [4]byte
	Revision             uint16
	Type                 uint16
	InfoOffset           int32
	InfoCount            int32
	HeaderSize           int32
	ItemCount            int32
	TotalSize            int64
	DataOffset           int64
	DataSize             int64
	ContentID            [36]byte
	_                    [12]byte
	Digest               [16]byte
	DataIV               [16]byte
	HeaderCmacHash       [16]byte
	HeaderNpdrmSignature [40]byte
	HeaderSha1Hash       [8]byte
}

func (h *FileHeader) GetContentID() string {
	return string(h.ContentID[:])
}

func (h *FileHeader) GetTitleID() string {
	return string(h.ContentID[7:16])
}

func (h *FileHeader) GetContentName() string {
	return h.GetContentID()[20:]
}

type ExtendedHeader struct {
	Magic       [4]byte
	Unknown1    uint32
	HeaderSize  int32
	DataSize    int32
	DataOffset  int32
	DataType    uint32
	PkgDataSize int64
	_           uint32
	DataType2   uint32
	Unknown2    uint32
	_           uint32
	_           uint64
	_           uint64
}

func (h *ExtendedHeader) KeyType() int {
	return int(h.DataType2) & 7
}

type Metadata struct {
	DrmType          uint32
	ContentType      ContentTypeEnum
	PackageFlags     uint32
	IndexTableOffset uint32
	IndexTableSize   uint32
	SfoOffset        uint32
	SfoSize          uint32
}

type ItemRecord struct {
	FilenameOffset uint32
	FilenameSize   int32
	DataOffset     int64
	DataSize       int64
	Flags          uint32
	Reserved       uint32
}

func (h *ItemRecord) KeyType() uint16 {
	flag := h.Flags >> 24 & 0xff
	return uint16(flag)
}

type InfoHeader struct {
	Type IdentifierType
	Size int32
}
