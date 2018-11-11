package main

import (
	"crypto/md5"
	"flag"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/keybase/kbfs/dokan"
)

//could be flags
var volumeinfo = dokan.VolumeInformation{
	VolumeName:             "cryptfs",
	VolumeSerialNumber:     0x1337,
	FileSystemName:         "cryptfs",
	MaximumComponentLength: 0xff,
	FileSystemFlags:        dokan.FileSupportsEncryption,
}

func main() {
	var dir, drive, key string
	var err error
	flag.StringVar(&dir, "dir", "", "Directory to mount")
	flag.StringVar(&drive, "drive", "Q:", "Directory to mount")
	flag.StringVar(&key, "key", "123456789qwertyu", "Directory to mount")
	flag.Parse()
	if dir == "" {
		dir, err = ioutil.TempDir("", "")
		if err != nil {
			log.Fatal("unable to create dir for mount storage: ", err)
		}
	}
	fmt.Println("Mount Dir: ", dir)
	var myFileSystem dokan.FileSystem = &cryptfs{root: dir, key: keyTo16Byte(key)}
	mp, err := dokan.Mount(&dokan.Config{FileSystem: myFileSystem, Path: drive})
	if err != nil {
		log.Fatal("Mount failed: ", err)
	}
	err = mp.BlockTillDone()
	if err != nil {
		log.Println("Filesystem exit: ", err)
	}
}

func keyTo16Byte(key string) []byte {
	v := md5.Sum([]byte(key))
	return v[:]
}
