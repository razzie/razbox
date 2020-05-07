package razlink

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-redis/redis/v7"
)

// DB ...
type DB struct {
	ExpirationTime time.Duration
	MaxLogs        int
	client         *redis.Client
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
		ExpirationTime: 30 * 24 * time.Hour, // ~1 month
		MaxLogs:        1000,
		client:         client,
	}, nil
}

// Close closes the connection to the database
func (db *DB) Close() error {
	return db.client.Close()
}

// InsertEntry inserts a new entry to the database
// If 'id' is null, the function will generate and return a unique one
func (db *DB) InsertEntry(id *string, e *Entry) (string, error) {
	if id == nil {
		tmpID := NewID()
		id = &tmpID
	}

	data, err := json.Marshal(e)
	if err != nil {
		return *id, err
	}

	expiration := db.ExpirationTime
	if e.Permanent {
		expiration = 0
	}

	success, err := db.client.SetNX(*id, string(data), expiration).Result()
	if err != nil {
		return *id, err
	}
	if !success {
		return *id, fmt.Errorf("duplicate ID: %s", *id)
	}

	return *id, nil
}

// SetEntry inserts or rewrites the entry with the given ID
func (db *DB) SetEntry(id string, e *Entry) error {
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}

	expiration := db.ExpirationTime
	if e.Permanent {
		expiration = 0
	}

	return db.client.Set(id, string(data), expiration).Err()
}

// GetEntry returns the entry with the given ID
func (db *DB) GetEntry(id string) (*Entry, error) {
	data, err := db.client.Get(id).Result()
	if err != nil {
		return nil, err
	}

	var e Entry
	err = json.Unmarshal([]byte(data), &e)
	if err != nil {
		return nil, err
	}

	// reset expiration
	if !e.Permanent {
		db.client.Expire(id, db.ExpirationTime)
	}
	db.client.Expire(id+"-log", db.ExpirationTime)

	return &e, nil
}

// GetEntries returns the list of entries with IDs matching the given pattern
func (db *DB) GetEntries(pattern string) (map[string]*Entry, error) {
	keys, err := db.client.Keys(pattern).Result()
	if err != nil {
		return nil, err
	}

	entries := make(map[string]*Entry)
	for _, id := range keys {
		if strings.HasSuffix(id, "-log") {
			continue
		}

		data, err := db.client.Get(id).Result()
		if err != nil {
			return nil, err
		}

		var e Entry
		err = json.Unmarshal([]byte(data), &e)
		if err != nil {
			return nil, err
		}

		entries[id] = &e
	}

	return entries, nil
}

// DeleteEntry deleted the entry with the given ID
func (db *DB) DeleteEntry(id string) error {
	defer db.DeleteLogs(id)
	return db.client.Del(id).Err()
}

// InsertLog inserts a new log
func (db *DB) InsertLog(entryID string, r *http.Request) error {
	l := NewLog(r)

	data, err := json.Marshal(l)
	if err != nil {
		return err
	}

	len, err := db.client.LPush(entryID+"-log", string(data)).Result()
	if err != nil {
		return err
	}

	if len == 1 {
		db.client.Expire(entryID+"-log", db.ExpirationTime)
	} else {
		db.client.LTrim(entryID+"-log", 0, int64(db.MaxLogs-1))
	}

	return nil
}

// GetLogs returns the Nth page of logs that belong to an entry (pages are 0 based)
func (db *DB) GetLogs(entryID string, first, last int) ([]Log, error) {
	values, err := db.client.LRange(entryID+"-log", int64(first), int64(last)).Result()
	if err != nil {
		return nil, err
	}

	var logs []Log
	for _, data := range values {
		var l Log
		err := json.Unmarshal([]byte(data), &l)
		if err == nil {
			logs = append(logs, l)
		}
	}

	return logs, nil
}

// GetLogsCount returns the number of logs that belong to an entry
func (db *DB) GetLogsCount(entryID string) (int, error) {
	value, err := db.client.LLen(entryID + "-log").Result()
	return int(value), err
}

// DeleteLogs deleted all logs that belong to an entry
func (db *DB) DeleteLogs(entryID string) error {
	return db.client.Del(entryID + "-log").Err()
}
