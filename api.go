package razbox

import (
	"path/filepath"

	"github.com/razzie/razbox/internal"
)

// API ...
type API struct {
	root string
	db   *internal.DB
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
		root: root,
	}, nil
}

// ConnectDB ...
func (api *API) ConnectDB(redisAddr, redisPw string, redisDb int) error {
	db, err := internal.NewDB(redisAddr, redisPw, redisDb)
	if err != nil {
		return err
	}

	api.db = db
	return nil
}
