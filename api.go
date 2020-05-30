package razbox

import (
	"path/filepath"
	"time"

	"github.com/razzie/razbox/internal"
)

// API ...
type API struct {
	root                string
	db                  *internal.DB
	CacheDuration       *time.Duration
	CookieExpiration    time.Duration
	ThumbnailRetryAfter time.Duration
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

	tmpCacheDuration := time.Hour
	return &API{
		root:                root,
		CacheDuration:       &tmpCacheDuration,
		CookieExpiration:    time.Hour * 24 * 7,
		ThumbnailRetryAfter: time.Hour,
	}, nil
}

// ConnectDB ...
func (api *API) ConnectDB(redisAddr, redisPw string, redisDb int) error {
	db, err := internal.NewDB(redisAddr, redisPw, redisDb)
	if err != nil {
		return err
	}

	db.CacheDuration = *api.CacheDuration
	api.db = db
	api.CacheDuration = &db.CacheDuration
	return nil
}
