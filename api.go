package razbox

import (
	"path/filepath"
	"time"

	"github.com/razzie/beepboop"
)

// API ...
type API struct {
	root                string
	db                  *beepboop.DB
	CacheDuration       *time.Duration
	CookieExpiration    *time.Duration
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

	tmpCookieExpiration := time.Hour * 24 * 7
	return &API{
		root:                root,
		CookieExpiration:    &tmpCookieExpiration,
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

	api.db = db
	api.CacheDuration = &db.CacheDuration
	api.CookieExpiration = &db.SessionDuration
	return db, nil
}
