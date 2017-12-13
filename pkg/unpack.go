package pkg

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
)

func loadSFO(w pkgWriter, pr *Reader, entry *fileEntry) error {
	sfo := bytes.Buffer{}
	sfoWriter := io.TeeReader(pr, &sfo)
	err := w.CreateFile(entry.Name, sfoWriter)
	_, err = pr.readSFO(&sfo)

	return err
}

func (pr *Reader) unpackLoop(w pkgWriter) error {
	for {
		entry, err := pr.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		switch {
		case entry.IsDirectory():
			err := w.CreateDir(entry.Name)
			if err != nil {
				return err
			}
		case entry.IsFile():
			if strings.HasSuffix(entry.Name, "PARAM.SFO") && len(pr.SfoEntries) == 0 {
				err = loadSFO(w, pr, entry)
			} else {
				err = w.CreateFile(entry.Name, pr)
			}

			if err != nil {
				return err
			}
		default:
			return errors.New("unknown file in package")
		}
	}

	if pr.pkgType != PackageTypeVitaDLC &&
		pr.pkgType != PackageTypeVitaApp &&
		pr.pkgType != PackageTypeVitaPatch {
		return nil
	}

	err := w.CreateDir("sce_sys/package")
	if err != nil {
		return err
	}

	err = w.CreateFile("sce_sys/package/head.bin", &pr.headBuffer)
	if err != nil {
		return err
	}

	err = w.CreateFile("sce_sys/package/tail.bin", &pr.tailBuffer)
	if err != nil {
		return err
	}

	err = w.CreateFile("sce_sys/package/work.bin", bytes.NewReader(pr.rif))
	if err != nil {
		return err
	}

	return nil
}

func (pr *Reader) Unpack(outDir string) error {
	titleid := pr.GetTitleID()

	var basedir string

	switch pr.PackageType() {
	case PackageTypeVitaApp:
		basedir = path.Join(outDir, "app", titleid)
	case PackageTypeVitaDLC:
		contentName := pr.FileHeader.GetContentName()
		basedir = path.Join(outDir, "cont", titleid, contentName)
	case PackageTypeVitaPatch:
		appVer := pr.SfoEntries["APP_VER"]
		appVer = strings.TrimLeft(appVer, "0")
		basedir = path.Join(outDir, "patch", titleid)
	case PackageTypePSP:
		basedir = path.Join(outDir, "pspemu/ISO")
	}

	err := os.MkdirAll(basedir, 0755)
	if err != nil {
		return err
	}

	return pr.unpackLoop(&fsPkgWriter{basedir: basedir})
}

func (pr *Reader) CreateZip(outDir string) error {
	title := pr.GetTitle()
	titleid := pr.GetTitleID()
	region := pr.GetRegion()

	var filename string
	var basedir string

	switch pr.PackageType() {
	case PackageTypeVitaApp:
		basedir = path.Join("app", titleid)
		filename = fmt.Sprintf("%s [%s] [%s].zip", title, titleid, region)
	case PackageTypeVitaDLC:
		contentName := pr.FileHeader.GetContentName()
		basedir = path.Join("cont", titleid, contentName)
		filename = fmt.Sprintf("%s [%s] [%s] [%s].zip", title, titleid, region, contentName)
	case PackageTypeVitaPatch:
		appVer := pr.SfoEntries["APP_VER"]
		appVer = strings.TrimLeft(appVer, "0")
		basedir = path.Join("patch", titleid)
		filename = fmt.Sprintf("%s [%s] [%s] [PATCH] [v%s].zip", title, titleid, region, appVer)
	case PackageTypePSP:
		basedir = path.Join("pspemu", titleid)
		if title == "" {
			filename = fmt.Sprintf("%s.zip", titleid)
		} else {
			filename = fmt.Sprintf("%s [%s] [%s].zip", title, titleid, region)
		}
	}

	filepath := path.Join(outDir, filename)

	if title == "" {
		// rename file after unpacking it, the internal sfo will be read then
		defer func(oldpath, titleid, region string) {
			title := pr.GetTitle()
			filename = fmt.Sprintf("%s [%s] [%s].zip", title, titleid, region)
			filepath := path.Join(outDir, filename)
			os.Rename(oldpath, filepath)
		}(filepath, titleid, region)
	}

	zf, err := os.Create(filepath)
	if err != nil {
		return err
	}

	defer zf.Close()

	zipWriter := zip.NewWriter(zf)
	defer zipWriter.Close()

	return pr.unpackLoop(&zipPkgWriter{zipWriter: zipWriter, basedir: basedir})
}
