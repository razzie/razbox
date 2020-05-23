package main

import (
	"flag"
	"log"
	"net/http"
	"strconv"

	"github.com/razzie/razbox/api"
)

// Command-line args
var (
	Root          string
	RedisAddr     string
	RedisPw       string
	RedisDb       int
	Port          int
	DefaultFolder string
)

func init() {
	flag.StringVar(&RedisAddr, "redis-addr", "localhost:6379", "Redis hostname:port")
	flag.StringVar(&RedisPw, "redis-pw", "", "Redis password")
	flag.IntVar(&RedisDb, "redis-db", 0, "Redis database (0-15)")
	flag.StringVar(&Root, "root", "./uploads", "Root directory of folders")
	flag.IntVar(&Port, "port", 8080, "HTTP port")
	flag.StringVar(&DefaultFolder, "default-folder", "", "Default folder to show in case of empty URL path")
}

func main() {
	flag.Parse()

	a, err := api.New(Root)
	if err != nil {
		log.Fatal(err)
	}

	err = a.ConnectDB(RedisAddr, RedisPw, RedisDb)
	if err != nil {
		log.Print("failed to connect to database:", err)
	}

	srv := NewServer(a, DefaultFolder)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(Port), srv))
}
