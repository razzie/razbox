package main

import (
	"flag"
	"log"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/razzie/razbox"
	"github.com/razzie/razlink"
)

var (
	// Port is the HTTP port of the application
	Port int
)

func main() {
	flag.StringVar(&razbox.Root, "root", "./uploads", "Root directory of folders")
	flag.IntVar(&Port, "port", 8080, "HTTP port")
	flag.Parse()

	if !filepath.IsAbs(razbox.Root) {
		var err error
		razbox.Root, err = filepath.Abs(razbox.Root)
		if err != nil {
			log.Fatal(err)
		}
	}

	srv := razlink.NewServer()
	srv.AddPages(&razbox.FolderPage)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(Port), srv))
}
