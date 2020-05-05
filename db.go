package razbox

import (
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v7"
)

// DB ...
type DB struct {
	client *redis.Client
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
		client: client,
	}, nil
}

// GetCachedFolder returns a cached Folder
func (db *DB) GetCachedFolder(path string) (*Folder, error) {
	data, err := db.client.Get(pathToKey(path)).Result()
	if err != nil {
		return nil, err
	}

	var folder Folder
	err = json.Unmarshal([]byte(data), &folder)
	if err != nil {
		return nil, err
	}

	return &folder, nil
}

// CacheFolder caches a Folder
func (db *DB) CacheFolder(folder *Folder) error {
	data, err := json.Marshal(folder)
	if err != nil {
		return err
	}

	return db.client.SetNX(pathToKey(folder.RelPath), string(data), time.Minute).Err()
}

func pathToKey(path string) string {
	if path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}
	return path
}
