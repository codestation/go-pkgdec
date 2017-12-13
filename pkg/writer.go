package pkg

import (
	"archive/zip"
	"io"
	"os"
	"path"
	"time"
)

type pkgWriter interface {
	CreateDir(path string) error
	CreateFile(path string, r io.Reader) error
}

type fsPkgWriter struct {
	basedir string
}

type zipPkgWriter struct {
	basedir   string
	zipWriter *zip.Writer
}

func (fs *fsPkgWriter) CreateDir(name string) error {
	fullPath := path.Join(fs.basedir, name)
	return os.MkdirAll(fullPath, 0755)
}

func (fs *fsPkgWriter) CreateFile(name string, r io.Reader) error {
	fullPath := path.Join(fs.basedir, name)
	pf, err := os.Create(fullPath)
	if err != nil {
		return err
	}

	defer pf.Close()

	if _, err = io.Copy(pf, r); err != nil {
		return err
	}

	return nil
}

func (fs *zipPkgWriter) CreateDir(name string) error {
	fullPath := path.Join(fs.basedir, name)
	header := &zip.FileHeader{
		Name:          fullPath + "/",
		ExternalAttrs: 0x10, // directory
	}

	header.SetModTime(time.Now())

	_, err := fs.zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	return nil
}

func (fs *zipPkgWriter) CreateFile(name string, r io.Reader) error {
	fullPath := path.Join(fs.basedir, name)
	header := &zip.FileHeader{
		Name: fullPath,
	}

	header.SetModTime(time.Now())
	header.Method = zip.Store

	pf, err := fs.zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	if _, err := io.Copy(pf, r); err != nil {
		return err
	}

	return nil
}
