package main

import (
	"encoding/base64"
	"flag"
	"log"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/razzie/razbox"
	"github.com/razzie/razlink"
)

// Command-line args
var (
	RedisAddr     string
	RedisPw       string
	RedisDb       int
	Port          int
	DefaultFolder string
)

func main() {
	flag.StringVar(&RedisAddr, "redis-addr", "localhost:6379", "Redis hostname:port")
	flag.StringVar(&RedisPw, "redis-pw", "", "Redis password")
	flag.IntVar(&RedisDb, "redis-db", 0, "Redis database (0-15)")
	flag.StringVar(&razbox.Root, "root", "./uploads", "Root directory of folders")
	flag.Int64Var(&razbox.DefaultMaxFileSizeMB, "max-file-size", 10, "Default file size limit for uploads in MiB")
	flag.IntVar(&Port, "port", 8080, "HTTP port")
	flag.StringVar(&DefaultFolder, "default-folder", "", "Default folder to show in case of empty URL path")
	flag.Parse()

	if !filepath.IsAbs(razbox.Root) {
		var err error
		razbox.Root, err = filepath.Abs(razbox.Root)
		if err != nil {
			log.Fatal(err)
		}
	}

	db, err := razbox.NewDB(RedisAddr, RedisPw, RedisDb)
	if err != nil {
		log.Print("failed to connect to database:", err)
	}

	srv := razlink.NewServer()
	srv.AddPages(
		GetStaticPage(),
		GetWelcomePage(DefaultFolder),
		GetFolderPage(db),
		GetReadAuthPage(db),
		GetWriteAuthPage(db),
		GetUploadPage(db),
		GetDownloadPage(db),
		GetEditPage(db),
		GetDeletePage(db),
		GetPasswordPage(db),
		GetGalleryPage(db),
		GetThumbnailPage(db),
		GetTextPage(db),
	)
	srv.FaviconPNG, _ = base64.StdEncoding.DecodeString("" +
		"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" +
		"AAAASUVORK5CYII=")
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(Port), srv))
}
