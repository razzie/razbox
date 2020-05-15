package main

import (
	"flag"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/razzie/razbox"
)

var (
	// SourceFile is the source file to be copied to the target folder
	SourceFile string
	// TargetFolder is the target/destination folder for the file
	TargetFolder string
	// Tags are the search tags for the file
	Tags string
)

func main() {
	flag.StringVar(&razbox.Root, "root", "./uploads", "Root directory of folders")
	flag.StringVar(&SourceFile, "file", "", "Source file to be copied to the target folder")
	flag.StringVar(&TargetFolder, "folder", "", "Target/destination folder for the file")
	flag.StringVar(&Tags, "tags", "", "Search tags for the file (space separated)")
	flag.Parse()

	file, err := os.Open(SourceFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	fi, _ := file.Stat()
	mime, _ := mimetype.DetectReader(file)
	file.Seek(0, io.SeekStart)

	basename := filepath.Base(SourceFile)
	boxfile := &razbox.File{
		Name:     basename,
		RelPath:  path.Join(TargetFolder, razbox.FilenameToUUID(basename)),
		Tags:     strings.Fields(Tags),
		MIME:     mime.String(),
		Size:     fi.Size(),
		Uploaded: fi.ModTime(),
	}

	err = boxfile.Create(file)
	if err != nil {
		log.Fatal(err)
	}
}
