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
	// Folder is the relative path of the folder compared to Root
	Folder string
)

func main() {
	flag.StringVar(&razbox.Root, "root", "./uploads", "Root directory of folders")
	flag.StringVar(&ReadPassword, "readpw", "", "Password for read access to the folder (optional)")
	flag.StringVar(&WritePassword, "writepw", "", "Password for write access to the folder")
	flag.StringVar(&Folder, "folder", "", "Folder name (relative path)")
	flag.Parse()

	if !filepath.IsAbs(razbox.Root) {
		var err error
		razbox.Root, err = filepath.Abs(razbox.Root)
		if err != nil {
			log.Fatal(err)
		}
	}

	err := os.MkdirAll(path.Join(razbox.Root, Folder), 0644)
	if err != nil {
		log.Fatal(err)
	}

	folder := &razbox.Folder{Path: path.Join(razbox.Root, Folder)}
	err = folder.SetPasswords(ReadPassword, WritePassword)
	if err != nil {
		log.Fatal(err)
	}
}
