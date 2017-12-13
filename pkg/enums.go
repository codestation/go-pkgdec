package pkg

const EntryTypePSP = 0x90

type IdentifierType uint32

const (
	IdentifierDRMType       IdentifierType = 0x1
	IdentifierContentType   IdentifierType = 0x2
	IdentifierPackageFlags  IdentifierType = 0x3
	IdentifierFileIndexInfo IdentifierType = 0xd
	IdentifierSFO           IdentifierType = 0xe
)

type ContentTypeEnum uint32

const (
	ContentTypePS1     ContentTypeEnum = 0x6
	ContentTypePSP     ContentTypeEnum = 0x7
	ContentTypePSPGo   ContentTypeEnum = 0xe
	ContentTypeMinis   ContentTypeEnum = 0xf
	ContentTypeNeoGeo  ContentTypeEnum = 0x10
	ContentTypeVitaApp ContentTypeEnum = 0x15
	ContentTypeVitaDLC ContentTypeEnum = 0x16
	ContentTypePSM1    ContentTypeEnum = 0x18
	ContentTypePSM2    ContentTypeEnum = 0x1c
)

type FileTypeEnum int

const (
	FileTypeFile0         FileTypeEnum = 0
	FileTypeFile1         FileTypeEnum = 1
	FileTypeFileEdat      FileTypeEnum = 2
	FileTypeFile3         FileTypeEnum = 3
	FileTypeDirectory     FileTypeEnum = 4
	FileTypeFileDocinfo   FileTypeEnum = 5
	FileTypeFilePbp       FileTypeEnum = 6
	FileTypeFileModule    FileTypeEnum = 14
	FileTypeFile15        FileTypeEnum = 15
	FileTypeFileKeystone  FileTypeEnum = 16
	FileTypeFilePfs       FileTypeEnum = 17
	FileTypeDirectoryPfs  FileTypeEnum = 18
	FileTypeFileTemp      FileTypeEnum = 19
	FileTypeFileInst      FileTypeEnum = 20
	FileTypeFileClearsign FileTypeEnum = 21
	FileTypeFileSys       FileTypeEnum = 22
	FileTypeFileDigs      FileTypeEnum = 24
)
