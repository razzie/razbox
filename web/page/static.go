package page

import (
	"github.com/razzie/beepboop"
)

// Static returns a beepboop.Page that handles static assets
func Static() *beepboop.Page {
	return beepboop.StaticAssetPage("/static/", "web/static")
}
