package main

import (
	"flag"
	"log"
	"time"

	"github.com/razzie/razbox"
)

// Command-line args
var (
	Root                string
	RedisAddr           string
	RedisPw             string
	RedisDb             int
	Port                int
	DefaultFolder       string
	CacheDuration       time.Duration
	CookieExpiration    time.Duration
	ThumbnailRetryAfter time.Duration
	AuthsPerMin         int
)

func init() {
	flag.StringVar(&RedisAddr, "redis-addr", "localhost:6379", "Redis hostname:port")
	flag.StringVar(&RedisPw, "redis-pw", "", "Redis password")
	flag.IntVar(&RedisDb, "redis-db", 0, "Redis database (0-15)")
	flag.StringVar(&Root, "root", "./uploads", "Root directory of folders")
	flag.IntVar(&Port, "port", 8080, "HTTP port")
	flag.StringVar(&DefaultFolder, "default-folder", "", "Default folder to show in case of empty URL path")
	flag.DurationVar(&CacheDuration, "cache-duration", time.Hour, "Cache duration")
	flag.DurationVar(&CookieExpiration, "cookie-expiration", time.Hour*24*7, "Cookie expiration for read and write access (1 week by default)")
	flag.DurationVar(&ThumbnailRetryAfter, "thumb-retry-after", time.Hour, "Duration to wait before attempting to create thumbnail again after fail")
	flag.IntVar(&AuthsPerMin, "auths-per-min", 3, "Max auth attempts/minute/IP (only works with Redis)")
	flag.Parse()
}

func main() {
	api, err := razbox.NewAPI(Root)
	if err != nil {
		log.Fatal(err)
	}

	// CacheDuration and CookieExpiration must be set before connecting to DB
	api.CacheDuration = CacheDuration
	api.CookieExpiration = CookieExpiration
	api.ThumbnailRetryAfter = ThumbnailRetryAfter
	api.AuthsPerMin = AuthsPerMin

	db, err := api.ConnectDB(RedisAddr, RedisPw, RedisDb)
	if err != nil {
		log.Print("failed to connect to database:", err)
	}

	srv := NewServer(api, DefaultFolder, db)
	log.Fatal(srv.Serve(Port))
}
