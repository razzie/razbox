package razbox

import (
	"path/filepath"
	"sync"
	"time"

	"github.com/razzie/beepboop"
)

// API ...
type API struct {
	root                string
	db                  *beepboop.DB
	folderLock          sync.Map
	CacheDuration       time.Duration
	CookieExpiration    time.Duration
	ThumbnailRetryAfter time.Duration
	AuthsPerMin         int
}

// NewAPI ...
func NewAPI(root string) (*API, error) {
	if !filepath.IsAbs(root) {
		var err error
		root, err = filepath.Abs(root)
		if err != nil {
			return nil, err
		}
	}

	return &API{
		root:                root,
		CacheDuration:       time.Hour,
		CookieExpiration:    time.Hour * 24 * 7,
		ThumbnailRetryAfter: time.Hour,
		AuthsPerMin:         3,
	}, nil
}

// ConnectDB ...
func (api *API) ConnectDB(redisAddr, redisPw string, redisDb int) (*beepboop.DB, error) {
	db, err := beepboop.NewDB(redisAddr, redisPw, redisDb)
	if err != nil {
		return nil, err
	}

	db.CacheDuration = api.CacheDuration
	db.SessionDuration = api.CookieExpiration
	api.db = db
	return db, nil
}
