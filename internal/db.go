package internal

import (
	"encoding/json"
	"path"
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
