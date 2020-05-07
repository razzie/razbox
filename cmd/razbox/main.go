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

// Command-line args
var (
	RedisAddr string
	RedisPw   string
	RedisDb   int
	Port      int
)

func main() {
	flag.StringVar(&RedisAddr, "redis-addr", "localhost:6379", "Redis hostname:port")
	flag.StringVar(&RedisPw, "redis-pw", "", "Redis password")
	flag.IntVar(&RedisDb, "redis-db", 0, "Redis database (0-15)")
	flag.StringVar(&razbox.Root, "root", "./uploads", "Root directory of folders")
	flag.Int64Var(&razbox.DefaultMaxFileSizeMB, "max-file-size", 10, "Default file size limit for uploads in MiB")
	flag.IntVar(&Port, "port", 8080, "HTTP port")
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
		&WelcomePage,
		GetFolderPage(db),
		GetSearchPage(db),
		GetReadAuthPage(db),
		GetWriteAuthPage(db),
		GetUploadPage(db),
		GetEditPage(db),
	)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(Port), srv))
}
