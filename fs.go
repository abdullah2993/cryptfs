package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/JamesHovious/w32"
	"github.com/abdullah2993/encfile"
	"github.com/keybase/kbfs/dokan"
)

type cryptfs struct {
	root string
	key  []byte
}

var _ dokan.FileSystem = (*cryptfs)(nil)

func (fs *cryptfs) WithContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return ctx, nil //because fuck context
}

func (fs *cryptfs) CreateFile(ctx context.Context, fi *dokan.FileInfo, data *dokan.CreateData) (file dokan.File, status dokan.CreateStatus, err error) {

	switch data.CreateDisposition {
	case dokan.FileOpen, dokan.FileOpenIf:
		pp := fs.getPath(fi.Path())
		fss, err := os.Stat(pp)
		if err != nil {
			err, _ = osErrToDokanErr(err, true)
			return nil, 0, err
		}
		if data.ReturningDirAllowed() == nil && fss.IsDir() {
			return newCryptFile(fs, fs.encFilePath(fi.Path()), nil), dokan.ExistingDir, nil
		} else if data.ReturningFileAllowed() == nil && !fss.IsDir() {
			// fd, err := os.OpenFile(pp, os.O_RDWR|os.O_APPEND, 666)
			fd, err := encfile.Append(pp, fs.key, 512)
			if err != nil {
				err, _ = osErrToDokanErr(err, true)
				return nil, 0, err
			}
			return newCryptFile(fs, fs.encFilePath(fi.Path()), fd), dokan.ExistingFile, nil
		}
	case dokan.FileCreate:
		pp := fs.getPath(fi.Path())
		_, err := os.Stat(pp)
		if err != nil && !os.IsNotExist(err) {
			err, _ = osErrToDokanErr(err, true)
			return nil, 0, err
		}
		if data.ReturningDirAllowed() == nil {

			err = os.Mkdir(pp, os.ModeDir)
			if err != nil {
				err, _ = osErrToDokanErr(err, true)
				return nil, 0, err
			}

			return newCryptFile(fs, fs.encFilePath(fi.Path()), nil), dokan.NewDir, nil

		} else if data.ReturningFileAllowed() == nil {

			// fd, err := os.OpenFile(pp, os.O_CREATE|os.O_RDWR, 666)
			fd, err := encfile.Create(pp, fs.key, 512)
			if err != nil {
				err, _ = osErrToDokanErr(err, false)
				return nil, 0, err
			}
			return newCryptFile(fs, fs.encFilePath(fi.Path()), fd), dokan.NewFile, nil

		}
	}

	return nil, 0, dokan.ErrNotSupported

}

func (fs *cryptfs) GetDiskFreeSpace(ctx context.Context) (dokan.FreeSpace, error) {
	fsi := dokan.FreeSpace{}
	r, freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes := w32.GetDiskFreeSpaceEx(fs.root)
	if r {
		fsi.FreeBytesAvailable = freeBytesAvailable
		fsi.TotalNumberOfBytes = totalNumberOfBytes
		fsi.TotalNumberOfFreeBytes = totalNumberOfFreeBytes
	}
	return fsi, nil
}

func (fs *cryptfs) GetVolumeInformation(ctx context.Context) (dokan.VolumeInformation, error) {
	return volumeinfo, nil
}

func (fs *cryptfs) MoveFile(ctx context.Context, sourceHandle dokan.File, sourceFileInfo *dokan.FileInfo, targetPath string, replaceExisting bool) error {
	return os.Rename(fs.getPath(sourceFileInfo.Path()), fs.getPath(targetPath))
	// panic("not implemented")
}

func (fs *cryptfs) ErrorPrint(err error) {
	fmt.Println(err)
}

func (fs *cryptfs) Printf(format string, v ...interface{}) {
	fmt.Printf(format, v...)
}

func (fs *cryptfs) getPath(p string) string {
	encFp := fs.encFilePath(p)
	return filepath.Join(fs.root, encFp)
}

func (fs *cryptfs) encFilePath(fp string) string {
	pp := strings.Split(fp, `\`)
	for i, v := range pp {
		cb, err := aes.NewCipher(fs.key)
		if err != nil {
			panic(err)
		}
		iv := make([]byte, 16)
		c := cipher.NewCTR(cb, iv)
		if err != nil {
			panic(err)
		}
		buff := make([]byte, len(v))
		c.XORKeyStream(buff, []byte(v))
		pp[i] = base64.URLEncoding.EncodeToString(buff)
	}
	return filepath.Join(pp...)
}

func (fs *cryptfs) decFilePath(fp string) string {
	pp := strings.Split(fp, `\`)
	for i, v := range pp {
		cb, err := aes.NewCipher(fs.key)
		if err != nil {
			panic(err)
		}
		iv := make([]byte, 16)
		c := cipher.NewCTR(cb, iv)
		if err != nil {
			panic(err)
		}
		bb, err := base64.URLEncoding.DecodeString(v)
		if err != nil {
			panic(err)
		}
		buff := make([]byte, len(bb))
		c.XORKeyStream(buff, []byte(bb))
		pp[i] = string(buff)
	}
	return filepath.Join(pp...)
}
