package pkg

import (
	"crypto/cipher"
	"errors"
	"io"
	"io/ioutil"
)

func (pr *Reader) Read(b []byte) (int, error) {
	if pr.err != nil {
		return 0, pr.err
	}
	if pr.current == nil {
		return 0, io.EOF
	}

	n, err := pr.current.Read(b)
	if err != nil && err != io.EOF {
		pr.err = err
	}
	return n, err
}

func (pr *Reader) Next() (*fileEntry, error) {
	if pr.err != nil {
		return nil, pr.err
	}
	hdr, err := pr.next()
	pr.err = err
	return hdr, err
}

func (pr *Reader) numBytes() int64 {
	if pr.current == nil {
		// No current file, so no bytes
		return 0
	}
	return pr.current.numBytes()
}

func (pr *Reader) skipUnread() error {
	dataSkip := pr.numBytes()
	totalSkip := dataSkip + pr.pad
	pr.current, pr.pad = nil, 0

	copySkipped, err := io.CopyN(ioutil.Discard, pr.reader, totalSkip)
	if err == io.EOF && copySkipped < dataSkip {
		err = io.ErrUnexpectedEOF
	}

	return err
}

func (pr *Reader) next() (*fileEntry, error) {
	if err := pr.skipUnread(); err != nil {
		return nil, err
	}

	entry, err := pr.readNextEntry()
	if err != nil {
		return nil, err
	}

	if err := pr.handleRegularFile(entry); err != nil {
		return nil, err
	}

	return entry, nil
}

func (pr *Reader) readNextEntry() (*fileEntry, error) {
	e := &pr.index

	if e.idx >= len(e.itemRecords) {
		err := pr.readTail()
		if err != nil {
			return nil, err
		} else {
			return nil, io.EOF
		}
	}

	entry := e.itemRecords[e.idx]
	e.idx++

	return &entry, nil
}

func (pr *Reader) handleRegularFile(entry *fileEntry) error {
	nb := entry.Size
	if entry.IsDirectory() {
		nb = 0
	}
	if nb < 0 {
		return errors.New("pkg: invalid pkg header")
	}

	e := &pr.index
	curr := e.itemRecords[e.idx-1].Offset

	var next int64
	if e.idx < len(e.itemRecords) {
		next = e.itemRecords[e.idx].Offset
	} else {
		next = pr.FileHeader.DataSize
	}

	r := NewCTR(entry.Key, pr.FileHeader.DataIV[:], entry.Offset/16)
	reader := cipher.StreamReader{S: r, R: pr.aesReader.RawReader()}

	pr.pad = next - curr - entry.Size
	pr.current = &regFileReader{r: reader, nb: nb}
	return nil
}
