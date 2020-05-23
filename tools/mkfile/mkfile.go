package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/razzie/razbox/lib"
)

var (
	// Root is the root directory of folders
	Root string
	// SourceFiles marks the source file(s) to be copied to the target folder
	SourceFiles string
	// TargetFolder is the target/destination folder for the file
	TargetFolder string
	// Tags are the search tags for the file
	Tags string
	// Move (if enabled) removes the original file(s)
	Move bool
)

func init() {
	flag.StringVar(&Root, "root", "./uploads", "Root directory of folders")
	flag.StringVar(&SourceFiles, "file", "", "Source file(s) to be copied to the target folder - supports patterns")
	flag.StringVar(&TargetFolder, "folder", "", "Relative path of target/destination folder for the file")
	flag.StringVar(&Tags, "tags", "", "Search tags for the file (space separated)")
	flag.BoolVar(&Move, "move", false, "Remove original file(s)")
}

func main() {
	flag.Parse()

	if len(SourceFiles) == 0 {
		log.Fatal("No source file(s)")
	}

	if len(TargetFolder) == 0 {
		log.Fatal("No target folder")
	}

	matches, err := filepath.Glob(SourceFiles)
	if err != nil {
		log.Fatal(err)
	}

	if len(matches) == 0 {
		log.Fatal("No matches for", SourceFiles)
	}

	for _, filename := range matches {
		basename := filepath.Base(filename)
		fmt.Printf("Creating file %s... ", path.Join(TargetFolder, basename))

		file, err := os.Open(filename)
		if err != nil {
			fmt.Println("error:", err)
			continue
		}

		fi, _ := file.Stat()
		mime, _ := mimetype.DetectReader(file)
		file.Seek(0, io.SeekStart)

		boxfile := &lib.File{
			Name:     basename,
			RelPath:  path.Join(TargetFolder, lib.FilenameToUUID(basename)),
			Tags:     strings.Fields(Tags),
			MIME:     mime.String(),
			Size:     fi.Size(),
			Uploaded: fi.ModTime(),
		}

		err = boxfile.Create(file, false)
		file.Close()
		if err != nil {
			fmt.Println("error:", err)
		} else {
			if Move {
				os.Remove(filename)
			}
			fmt.Println("done")
		}
	}
}
