package main

import (
	"flag"
	"log"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/razzie/razbox/lib"
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
	flag.StringVar(&lib.Root, "root", "./uploads", "Root directory of folders")
	flag.Int64Var(&lib.DefaultMaxFileSizeMB, "max-file-size", 10, "Default file size limit for uploads in MiB")
	flag.IntVar(&Port, "port", 8080, "HTTP port")
	flag.StringVar(&DefaultFolder, "default-folder", "", "Default folder to show in case of empty URL path")
	flag.Parse()

	if !filepath.IsAbs(lib.Root) {
		var err error
		lib.Root, err = filepath.Abs(lib.Root)
		if err != nil {
			log.Fatal(err)
		}
	}

	db, err := lib.NewDB(RedisAddr, RedisPw, RedisDb)
	if err != nil {
		log.Print("failed to connect to database:", err)
	}

	srv := NewServer(db)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(Port), srv))
}
