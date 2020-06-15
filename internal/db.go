package internal

import (
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/go-redis/redis/v7"
)

// DB ...
type DB struct {
	client        *redis.Client
	CacheDuration time.Duration
}

// NewDB returns a new DB
func NewDB(addr, password string, db int) (*DB, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	err := client.Ping().Err()
	if err != nil {
		client.Close()
		return nil, err
	}

	return &DB{
		client:        client,
		CacheDuration: time.Hour,
	}, nil
}

// GetCachedFolder returns a cached Folder
func (db *DB) GetCachedFolder(folderName string) (*Folder, error) {
	data, err := db.client.Get("folder:" + path.Clean(folderName)).Result()
	if err != nil {
		return nil, err
	}

	var folder Folder
	err = json.Unmarshal([]byte(data), &folder)
	if err != nil {
		return nil, err
	}

	if len(folder.CachedFiles) == 0 || len(folder.CachedSubfolders) == 0 {
		return nil, &ErrFolderMissingCache{Folder: folder.RelPath}
	}

	return &folder, nil
}

// CacheFolder caches a Folder
func (db *DB) CacheFolder(folder *Folder) error {
	if len(folder.CachedFiles) == 0 {
		folder.GetFiles()
	}
	if len(folder.CachedSubfolders) == 0 {
		folder.GetSubfolders()
	}

	data, err := json.Marshal(folder)
	if err != nil {
		return err
	}

	return db.client.Set("folder:"+path.Clean(folder.RelPath), string(data), db.CacheDuration).Err()
}

// UncacheFolder uncaches a Folder
func (db *DB) UncacheFolder(folderName string) error {
	return db.client.Del("folder:" + path.Clean(folderName)).Err()
}

// IsWithinRateLimit returns whether a request is withing rate limit per minute
func (db *DB) IsWithinRateLimit(reqType, ip string, rate int) (bool, error) {
	key := fmt.Sprintf("rate:%s:%s", reqType, ip)
	pipe := db.client.TxPipeline()
	incr := pipe.Incr(key)
	pipe.Expire(key, time.Minute)
	_, err := pipe.Exec()
	if err != nil {
		return false, err
	}

	return int(incr.Val()) <= rate, nil
}

// GetSessionToken returns an access token from session ID
func (db *DB) GetSessionToken(sessionID string) (*AccessToken, error) {
	token := &AccessToken{
		Read:  make(map[string]string),
		Write: make(map[string]string),
	}
	return token, db.FillSessionToken(sessionID, token)
}

// FillSessionToken fills an existing token with extra access data from session ID
func (db *DB) FillSessionToken(sessionID string, token *AccessToken) error {
	reads, err := db.client.SMembers("session-read:" + sessionID).Result()
	if err != nil {
		return err
	}
	writes, err := db.client.SMembers("session-write:" + sessionID).Result()
	if err != nil {
		return err
	}

	for _, read := range reads {
		parts := strings.SplitN(read, ":", 2) // folderhash:passwordhash
		token.Read[parts[0]] = parts[1]
	}
	for _, write := range writes {
		parts := strings.SplitN(write, ":", 2) // folderhash:passwordhash
		token.Write[parts[0]] = parts[1]
	}
	return nil
}

// AddSessionToken adds an access token to an existing session
// (or creates a new session if didn't exist)
func (db *DB) AddSessionToken(sessionID string, token *AccessToken, expiration time.Duration) error {
	var reads, writes []interface{}
	for folderhash, pwhash := range token.Read {
		reads = append(reads, fmt.Sprintf("%s:%s", folderhash, pwhash))
	}
	for folderhash, pwhash := range token.Write {
		writes = append(writes, fmt.Sprintf("%s:%s", folderhash, pwhash))
	}

	pipe := db.client.TxPipeline()
	if len(reads) > 0 {
		pipe.SAdd("session-read:"+sessionID, reads...)
	}
	if len(writes) > 0 {
		pipe.SAdd("session-write:"+sessionID, writes...)
	}
	pipe.Expire("session-read:"+sessionID, expiration)
	pipe.Expire("session-write:"+sessionID, expiration)
	_, err := pipe.Exec()
	return err
}
