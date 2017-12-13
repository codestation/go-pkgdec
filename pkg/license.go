package pkg

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"errors"
	"io/ioutil"
)

const rifDictBase64 = `
eNpjYBgFo2AU0AsYAIElGt8MRJiDCAsw3xhEmIAIU4N4AwNdRxcXZ3+/EJCAkW
6Ac7C7ARwYgviuQAaIdoPSzlDaBUo7QmknIM3ACIZM78+u7kx3VWYEAGJ9HV0=
`

var rifDict = expandZlibDict()

func expandZlibDict() []byte {
	compressedDict, err := base64.StdEncoding.DecodeString(rifDictBase64)
	if err != nil {
		panic(err)
	}

	b := bytes.NewReader(compressedDict)
	z, err := zlib.NewReader(b)
	if err != nil {
		panic(err)
	}

	defer z.Close()

	dict, err := ioutil.ReadAll(z)
	if err != nil {
		panic(err)
	}

	return dict
}

func licenseSize(pkgType PackageType) int {
	switch pkgType {
	case PackageTypePSM:
		return 1024
	case PackageTypeVitaApp:
		fallthrough
	case PackageTypeVitaDLC:
		fallthrough
	case PackageTypeVitaPatch:
		return 512
	default:
		return 0
	}
}

func DecodeLicense(src string, pkgType PackageType) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(src)
	if err != nil {
		return nil, err
	}

	b := bytes.NewReader(data)
	z, err := zlib.NewReaderDict(b, rifDict)
	if err != nil {
		return nil, err
	}

	defer z.Close()
	lic, err := ioutil.ReadAll(z)
	if err != nil {
		return nil, err
	}

	if pkgType > 0 && len(lic) != licenseSize(pkgType) {
		return nil, errors.New("invalid license length")
	}

	return lic, nil
}

func EncodeLicense(data []byte) (string, error) {
	var b bytes.Buffer

	z, err := zlib.NewWriterLevelDict(&b, zlib.BestCompression, rifDict)

	if err != nil {
		return "", err
	}

	_, err = z.Write(data)
	if err != nil {
		return "", err
	}

	z.Close()

	compressedDict := b.Bytes()

	// fix the header to match other zRIF apps
	compressedDict[0] = 8       // CM = DEFLATE
	compressedDict[0] |= 2 << 4 // CINFO = 2 (1024 window size)
	compressedDict[1] = 3 << 6  // FLEVEL = 3 (max compression)
	compressedDict[1] |= 1 << 5 // FDICT = 1 (present)
	// FCHECK
	compressedDict[1] += uint8(31 - (uint16(compressedDict[0])<<8+uint16(compressedDict[1]))%31)

	return base64.StdEncoding.EncodeToString(compressedDict), nil
}
