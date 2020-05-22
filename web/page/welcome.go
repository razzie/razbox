package page

import (
	"fmt"
	"net/http"

	"github.com/razzie/razlink"
)

// Welcome returns a razlink.Page that redirects the visitor to the default folder
func Welcome(defaultFolder string) *razlink.Page {
	return &razlink.Page{
		Path: "/",
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return razlink.RedirectView(r, fmt.Sprintf("/x/%s/%s", defaultFolder, r.URL.RequestURI()))
		},
	}
}
