package main

import (
	"flag"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/razzie/razbox"
)

// Command-line args
var (
	Root          string
	RedisAddr     string
	RedisPw       string
	RedisDb       int
	Port          int
	DefaultFolder string
	CacheDuration time.Duration
)

func init() {
	flag.StringVar(&RedisAddr, "redis-addr", "localhost:6379", "Redis hostname:port")
	flag.StringVar(&RedisPw, "redis-pw", "", "Redis password")
	flag.IntVar(&RedisDb, "redis-db", 0, "Redis database (0-15)")
	flag.StringVar(&Root, "root", "./uploads", "Root directory of folders")
	flag.IntVar(&Port, "port", 8080, "HTTP port")
	flag.StringVar(&DefaultFolder, "default-folder", "", "Default folder to show in case of empty URL path")
	flag.DurationVar(&CacheDuration, "cache-duration", time.Hour, "Cache duration")
}

func main() {
	flag.Parse()

	api, err := razbox.NewAPI(Root)
	if err != nil {
		log.Fatal(err)
	}

	err = api.ConnectDB(RedisAddr, RedisPw, RedisDb)
	if err != nil {
		log.Print("failed to connect to database:", err)
	}

	if api.CacheDuration != nil {
		*api.CacheDuration = CacheDuration
	}

	srv := NewServer(api, DefaultFolder)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(Port), srv))
}
