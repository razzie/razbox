package page

import (
	"github.com/razzie/beepboop"
	"github.com/razzie/razbox/web"
)

// Static returns a beepboop.Page that handles static assets
func Static() *beepboop.Page {
	return beepboop.FSPage("/static/", web.Assets)
}
