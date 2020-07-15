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
func (db *DB) GetSessionToken(sessionID, ip string) (*AccessToken, error) {
	token := &AccessToken{
		Read:  make(map[string]string),
		Write: make(map[string]string),
	}
	return token, db.FillSessionToken(sessionID, ip, token)
}

// FillSessionToken fills an existing token with extra access data from session ID
func (db *DB) FillSessionToken(sessionID, ip string, token *AccessToken) error {
	key := fmt.Sprintf("%s:%s", sessionID, ip)
	reads, err := db.client.SMembers("session-read:" + key).Result()
	if err != nil {
		return err
	}
	writes, err := db.client.SMembers("session-write:" + key).Result()
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
func (db *DB) AddSessionToken(sessionID, ip string, token *AccessToken, expiration time.Duration) error {
	key := fmt.Sprintf("%s:%s", sessionID, ip)
	var reads, writes []interface{}
	for folderhash, pwhash := range token.Read {
		reads = append(reads, fmt.Sprintf("%s:%s", folderhash, pwhash))
	}
	for folderhash, pwhash := range token.Write {
		writes = append(writes, fmt.Sprintf("%s:%s", folderhash, pwhash))
	}

	pipe := db.client.TxPipeline()
	if len(reads) > 0 {
		pipe.SAdd("session-read:"+key, reads...)
	}
	if len(writes) > 0 {
		pipe.SAdd("session-write:"+key, writes...)
	}
	pipe.Expire("session-read:"+key, expiration)
	pipe.Expire("session-write:"+key, expiration)
	_, err := pipe.Exec()
	return err
}

// RemoveSessionToken removes an access token from an existing session
// (typically due to password change)
func (db *DB) RemoveSessionToken(sessionID, ip string, token *AccessToken) error {
	key := fmt.Sprintf("%s:%s", sessionID, ip)
	var reads, writes []interface{}
	for folderhash, pwhash := range token.Read {
		reads = append(reads, fmt.Sprintf("%s:%s", folderhash, pwhash))
	}
	for folderhash, pwhash := range token.Write {
		writes = append(writes, fmt.Sprintf("%s:%s", folderhash, pwhash))
	}

	pipe := db.client.TxPipeline()
	if len(reads) > 0 {
		pipe.SRem("session-read:"+key, reads...)
	}
	if len(writes) > 0 {
		pipe.SRem("session-write:"+key, writes...)
	}
	_, err := pipe.Exec()
	return err
}
