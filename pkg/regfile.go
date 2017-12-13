package pkg

import "io"

// A numBytesReader is an io.Reader with a numBytes method, returning the number
// of bytes remaining in the underlying encoded data.
type numBytesReader interface {
	io.Reader
	numBytes() int64
}

// A regFileReader is a numBytesReader for reading file data from a pkg archive.
type regFileReader struct {
	r  io.Reader
	nb int64
}

func (rfr *regFileReader) Read(b []byte) (n int, err error) {
	if rfr.nb == 0 {
		// file consumed
		return 0, io.EOF
	}
	if int64(len(b)) > rfr.nb {
		b = b[0:rfr.nb]
	}
	n, err = rfr.r.Read(b)
	rfr.nb -= int64(n)

	if err == io.EOF && rfr.nb > 0 {
		err = io.ErrUnexpectedEOF
	}
	return
}

// numBytes returns the number of bytes left to read in the file's data in the pkg archive.
func (rfr *regFileReader) numBytes() int64 {
	return rfr.nb
}
