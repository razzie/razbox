package main

import (
	"flag"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/razzie/razbox"
)

var (
	// ReadPassword is the read password for the target folder
	ReadPassword string
	// WritePassword is the write password for the target folder
	WritePassword string
	// MaxFileSizeMB is the upload file size limit in MiB for the folder
	MaxFileSizeMB int64
	// Folder is the relative path of the folder compared to Root
	Folder string
)

func main() {
	flag.StringVar(&razbox.Root, "root", "./uploads", "Root directory of folders")
	flag.StringVar(&ReadPassword, "readpw", "", "Password for read access to the folder (optional)")
	flag.StringVar(&WritePassword, "writepw", "", "Password for write access to the folder")
	flag.Int64Var(&MaxFileSizeMB, "max-file-size", 10, "File size limit for uploads in MiB for this folder")
	flag.StringVar(&Folder, "folder", "", "Folder name (relative path)")
	flag.Parse()

	if !filepath.IsAbs(razbox.Root) {
		var err error
		razbox.Root, err = filepath.Abs(razbox.Root)
		if err != nil {
			log.Fatal(err)
		}
	}

	err := os.MkdirAll(path.Join(razbox.Root, Folder), 0755)
	if err != nil {
		log.Fatal(err)
	}

	folder := &razbox.Folder{
		RelPath:       Folder,
		MaxFileSizeMB: MaxFileSizeMB,
	}
	err = folder.SetPasswords(ReadPassword, WritePassword)
	if err != nil {
		log.Fatal(err)
	}
}
