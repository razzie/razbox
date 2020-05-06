package main

import (
	"flag"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

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

	src, err := os.Open(SourceFile)
	if err != nil {
		log.Fatal(err)
	}
	defer src.Close()

	srci, _ := src.Stat()
	mime, _ := mimetype.DetectReader(src)
	src.Seek(0, io.SeekStart)

	basename := filepath.Base(SourceFile)
	dst := &razbox.File{
		Name:         basename,
		InternalName: path.Join(razbox.Root, TargetFolder, razbox.FilenameToUUID(basename)),
		Tags:         strings.Fields(Tags),
		MIME:         mime.String(),
		Size:         razbox.ByteCountSI(srci.Size()),
		Uploaded:     time.Now(),
	}

	err = dst.Save(src)
	if err != nil {
		log.Fatal(err)
	}
}
