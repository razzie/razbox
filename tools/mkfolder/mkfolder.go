package main

import (
	"flag"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/razzie/razbox/lib"
)

var (
	// Root is the root directory of folders
	Root string
	// ReadPassword is the read password for the target folder
	ReadPassword string
	// WritePassword is the write password for the target folder
	WritePassword string
	// MaxFileSizeMB is the upload file size limit in MiB for the folder
	MaxFileSizeMB int64
	// Folder is the relative path of the folder compared to Root
	Folder string
)

func init() {
	flag.StringVar(&Root, "root", "./uploads", "Root directory of folders")
	flag.StringVar(&ReadPassword, "readpw", "", "Password for read access to the folder (optional)")
	flag.StringVar(&WritePassword, "writepw", "", "Password for write access to the folder")
	flag.Int64Var(&MaxFileSizeMB, "max-file-size", 10, "File size limit for uploads in MiB for this folder")
	flag.StringVar(&Folder, "folder", "", "Folder name (relative path)")
}

func main() {
	flag.Parse()

	if !filepath.IsAbs(Root) {
		var err error
		Root, err = filepath.Abs(Root)
		if err != nil {
			log.Fatal(err)
		}
	}

	err := os.MkdirAll(path.Join(Root, Folder), 0755)
	if err != nil {
		log.Fatal(err)
	}

	folder := &lib.Folder{
		RelPath:       Folder,
		MaxFileSizeMB: MaxFileSizeMB,
	}
	err = folder.SetPasswords(ReadPassword, WritePassword)
	if err != nil {
		log.Fatal(err)
	}
}
