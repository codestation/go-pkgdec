package pkg

import (
	"crypto/aes"
	"crypto/cipher"
	"io"
)

type ctrStream struct {
	block  cipher.Block
	stream cipher.Stream
	reader cipher.StreamReader
	iv     []byte
}

func (r *ctrStream) XORKeyStream(dst, src []byte) {
	r.stream.XORKeyStream(dst, src)
}

func (r *ctrStream) RawReader() io.Reader {
	return r.reader.R
}

func (r *ctrStream) SetRawReader(reader io.Reader) {
	r.reader.R = reader
}

func (r *ctrStream) Read(dst []byte) (n int, err error) {
	return r.reader.Read(dst)
}

func (r *ctrStream) SetCounter(counter int64) {
	r.stream = NewCTR(r.block, r.iv, counter)
	reader := r.reader.R
	r.reader = cipher.StreamReader{R: reader, S: r.stream}
}

func NewCTRReader(r io.Reader, key, iv []byte, counter int64) (*ctrStream, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	stream := NewCTR(block, iv, counter)

	return &ctrStream{
		block:  block,
		stream: stream,
		reader: cipher.StreamReader{R: r, S: stream},
		iv:     dup(iv),
	}, nil
}

func NewCTR(b cipher.Block, iv []byte, counter int64) cipher.Stream {
	var newIV []byte
	if counter > 0 {
		newIV = addCTRCounter(iv, counter)
	} else {
		newIV = iv
	}

	return cipher.NewCTR(b, newIV)
}

func AESCTRDecrypt(block cipher.Block, dst, src, iv []byte, counter int64) {
	ctr := NewCTR(block, iv, counter)
	ctr.XORKeyStream(dst, src)
}

func AESECBEncrypt(dst, src, key []byte) error {
	c, err := aes.NewCipher(key)
	if err == nil {
		c.Encrypt(dst, src)
		return nil
	}

	return err
}

func addCTRCounter(iv []byte, value int64) []byte {
	ctr := dup(iv)
	n := 16

	for {
		n--
		value += int64(ctr[n])
		ctr[n] = byte(value)
		value >>= 8

		if n == 0 {
			break
		}
	}

	return ctr
}

func dup(p []byte) []byte {
	q := make([]byte, len(p))
	copy(q, p)
	return q
}
