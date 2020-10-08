package beepboop

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v7"
)

// DB ...
type DB struct {
	client          *redis.Client
	CacheDuration   time.Duration
	SessionDuration time.Duration
}

type dbContextKeyType struct{}

var dbContextKey = &dbContextKeyType{}

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
		client:          client,
		CacheDuration:   time.Hour,
		SessionDuration: time.Hour * 24 * 7,
	}, nil
}

// CacheValue caches a value
func (db *DB) CacheValue(key string, value interface{}, rewriteExisting bool) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	if rewriteExisting {
		return db.client.Set("beepboop-cache:"+key, data, db.CacheDuration).Err()
	}
	return db.client.SetNX("beepboop-cache:"+key, data, db.CacheDuration).Err()
}

// UncacheValue removes a cached value
func (db *DB) UncacheValue(key string) error {
	return db.client.Del("beepboop-cache:" + key).Err()
}

// GetCachedValue tries to unmarshal a cached value
func (db *DB) GetCachedValue(key string, value interface{}) error {
	data, err := db.client.Get("beepboop-cache:" + key).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(data), value)
}

// IsWithinRateLimit returns whether a request is withing rate limit per minute
func (db *DB) IsWithinRateLimit(reqType, ip string, rate int) (bool, error) {
	key := fmt.Sprintf("beepboop-rate:%s:%s", reqType, ip)
	pipe := db.client.TxPipeline()
	incr := pipe.Incr(key)
	pipe.Expire(key, time.Minute)
	_, err := pipe.Exec()
	if err != nil {
		return false, err
	}

	return int(incr.Val()) <= rate, nil
}

// AddSessionAccess merges an AccessMap into a session's AccessMap
func (db *DB) AddSessionAccess(sessionID, ip string, access AccessMap) error {
	key := fmt.Sprintf("beepboop-session:%s:%s", sessionID, ip)
	data, _ := db.client.Get(key).Result()

	var sessAccess AccessMap
	if len(data) > 0 {
		err := json.Unmarshal([]byte(data), &sessAccess)
		if err != nil {
			return err
		}
		sessAccess.Merge(access)
	} else {
		sessAccess = access
	}

	newData, err := json.Marshal(sessAccess)
	if err != nil {
		return err
	}

	return db.client.Set(key, string(newData), db.SessionDuration).Err()
}

// RemoveSessionAccess removes/unmerges an AccessMap from a session's AccessMap
func (db *DB) RemoveSessionAccess(sessionID, ip string, access AccessMap) error {
	key := fmt.Sprintf("beepboop-session:%s:%s", sessionID, ip)
	data, err := db.client.Get(key).Result()
	if err != nil {
		return err
	}

	sessAccess := make(AccessMap)
	err = json.Unmarshal([]byte(data), &sessAccess)
	if err != nil {
		return err
	}

	sessAccess.Unmerge(access)

	newData, err := json.Marshal(sessAccess)
	if err != nil {
		return err
	}

	return db.client.Set(key, string(newData), db.SessionDuration).Err()
}

// GetAccessToken returns an AccessToken for the given session
func (db *DB) GetAccessToken(sessionID, ip string) (*AccessToken, error) {
	key := fmt.Sprintf("beepboop-session:%s:%s", sessionID, ip)
	data, err := db.client.Get(key).Result()
	if err != nil {
		return nil, err
	}

	var access AccessMap
	err = json.Unmarshal([]byte(data), &access)
	if err != nil {
		return nil, err
	}

	return &AccessToken{
		SessionID: sessionID,
		IP:        ip,
		AccessMap: access,
	}, nil
}
