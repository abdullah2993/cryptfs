package main

import (
	"os"
	"syscall"
	"time"

	"github.com/keybase/kbfs/dokan"
)

func has(val, flag uint32) bool {
	return val&flag != 0
}

func hasAttribute(val, flag dokan.FileAttribute) bool {
	return val&flag != 0
}

func fileInfoToStat(fi os.FileInfo) dokan.Stat {
	fstat := fi.Sys().(*syscall.Win32FileAttributeData)
	lastAccess := time.Unix(0, fstat.LastAccessTime.Nanoseconds())
	lastWrite := time.Unix(0, fstat.LastWriteTime.Nanoseconds())
	createdTime := time.Unix(0, fstat.CreationTime.Nanoseconds())
	c := dokan.Stat{
		Creation:           createdTime,
		FileSize:           fi.Size(),
		LastAccess:         lastAccess,
		LastWrite:          lastWrite,
		VolumeSerialNumber: 0x1337,
		FileAttributes:     dokan.FileAttribute(fstat.FileAttributes),
	}
	if fi.IsDir() {
		c.FileAttributes |= dokan.FileAttributeDirectory
	}

	return c
}

func osErrToDokanErr(err error, dir bool) (error, bool) {
	if os.IsExist(err) {
		if dir {
			return dokan.ErrObjectNameCollision, true
		}
		return dokan.ErrFileAlreadyExists, true
	}

	if os.IsNotExist(err) {
		if dir {
			return dokan.ErrObjectPathNotFound, true
		}
		return dokan.ErrObjectNameNotFound, true
	}

	if os.IsPermission(err) {
		return dokan.ErrAccessDenied, true
	}

	return err, false
}
