package razbox

import (
	"net/http"

	"github.com/razzie/razlink"
)

// WelcomePage is the razbox landing page
var WelcomePage = razlink.Page{
	Path: "/",
	Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
		return razlink.RedirectView(r, "/x/")
	},
}
