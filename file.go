package main

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/JonathanLogan/encfile"
	"github.com/keybase/kbfs/dokan"
	"github.com/keybase/kbfs/dokan/winacl"
)

type cryptfile struct {
	path     string
	fs       *cryptfs
	fullpath string
	isdir    bool
	f        *encfile.EncryptedFile
}

var currentUserSID, _ = winacl.CurrentProcessUserSid()
var currentGroupSID, _ = winacl.CurrentProcessPrimaryGroupSid()

var _ dokan.File = (*cryptfile)(nil)

func newCryptFile(fs *cryptfs, fp string, f *encfile.EncryptedFile) *cryptfile {
	c := &cryptfile{
		path:     fp,
		fs:       fs,
		fullpath: filepath.Join(fs.root, fp),
		isdir:    f == nil,
		f:        f,
	}
	return c
}

func (f *cryptfile) ReadFile(ctx context.Context, fi *dokan.FileInfo, bs []byte, offset int64) (int, error) {
	if !f.isdir {
		return f.f.ReadAt(bs, offset)
	}
	panic("dir read")
}

func (f *cryptfile) WriteFile(ctx context.Context, fi *dokan.FileInfo, bs []byte, offset int64) (int, error) {
	if !f.isdir {
		return f.f.WriteAt(bs, offset)
	}
	panic("dir write")
}

func (f *cryptfile) FlushFileBuffers(ctx context.Context, fi *dokan.FileInfo) error {
	if !f.isdir {
		return f.f.Sync()
	}
	panic("dir flush")
}

func (f *cryptfile) GetFileInformation(ctx context.Context, fi *dokan.FileInfo) (*dokan.Stat, error) {
	fistat, err := os.Stat(f.fullpath)
	if err != nil {
		return nil, err
	}
	stat := fileInfoToStat(fistat)
	return &stat, err
}

func (f *cryptfile) FindFiles(ctx context.Context, fi *dokan.FileInfo, pattern string, fillStatCallback func(*dokan.NamedStat) error) error {
	if f.isdir {
		files, err := ioutil.ReadDir(f.fullpath)
		if err != nil {
			return dokan.ErrAccessDenied
		}
		for _, fv := range files {
			fillStatCallback(&dokan.NamedStat{
				Name: f.fs.decFilePath(fv.Name()),
				Stat: fileInfoToStat(fv),
			})
		}
		return nil
	}
	return dokan.ErrNotADirectory
}

func (f *cryptfile) SetFileTime(ctx context.Context, fi *dokan.FileInfo, creation time.Time, lastAccess time.Time, lastWrite time.Time) error {
	return os.Chtimes(f.fullpath, lastAccess, lastWrite)
}

func (f *cryptfile) SetFileAttributes(ctx context.Context, fi *dokan.FileInfo, fileAttributes dokan.FileAttribute) error {
	return syscall.SetFileAttributes(syscall.StringToUTF16Ptr(f.fullpath), uint32(fileAttributes))
}

func (f *cryptfile) SetEndOfFile(ctx context.Context, fi *dokan.FileInfo, length int64) error {
	if !f.isdir {
		// return f.f.Truncate(length)
		//SetEndOfFile has a syscall but I think it works just like Truncate
		return nil
	}
	panic("dir truncate")
}

func (f *cryptfile) SetAllocationSize(ctx context.Context, fi *dokan.FileInfo, length int64) error {
	if !f.isdir {
		// return f.f.Truncate(length)
		//encfile needs support for Truncate
		return nil
	}
	return nil
}

func (f *cryptfile) LockFile(ctx context.Context, fi *dokan.FileInfo, offset int64, length int64) error {
	panic("not implemented")
	//no idea about it can use flock or just a mutex????
}

func (f *cryptfile) UnlockFile(ctx context.Context, fi *dokan.FileInfo, offset int64, length int64) error {
	panic("not implemented")
}

func (f *cryptfile) GetFileSecurity(ctx context.Context, fi *dokan.FileInfo, si winacl.SecurityInformation, sd *winacl.SecurityDescriptor) error {
	if si&winacl.OwnerSecurityInformation != 0 && currentUserSID != nil {
		sd.SetOwner(currentUserSID)
	}
	if si&winacl.GroupSecurityInformation != 0 && currentGroupSID != nil {
		sd.SetGroup(currentGroupSID)
	}
	if si&winacl.DACLSecurityInformation != 0 {
		var acl winacl.ACL
		acl.AddAllowAccess(0x001F01FF, currentUserSID)
		sd.SetDacl(&acl)
	}
	return nil
}

func (f *cryptfile) SetFileSecurity(ctx context.Context, fi *dokan.FileInfo, si winacl.SecurityInformation, sd *winacl.SecurityDescriptor) error {
	if si&winacl.OwnerSecurityInformation != 0 && currentUserSID != nil {
		sd.SetOwner(currentUserSID)
	}
	if si&winacl.GroupSecurityInformation != 0 && currentGroupSID != nil {
		sd.SetGroup(currentGroupSID)
	}
	if si&winacl.DACLSecurityInformation != 0 {
		var acl winacl.ACL
		acl.AddAllowAccess(0x001F01FF, currentUserSID)
		sd.SetDacl(&acl)
	}
	return nil
}

func (f *cryptfile) CanDeleteFile(ctx context.Context, fi *dokan.FileInfo) error {
	return nil
}

func (f *cryptfile) CanDeleteDirectory(ctx context.Context, fi *dokan.FileInfo) error {
	return nil
}

func (f *cryptfile) Cleanup(ctx context.Context, fi *dokan.FileInfo) {
	f.CloseFile(ctx, fi)
	if fi.IsDeleteOnClose() {
		os.RemoveAll(f.fullpath)
	}
}

func (f *cryptfile) CloseFile(ctx context.Context, fi *dokan.FileInfo) {
	if !f.isdir {
		f.f.Close()
	}
}
