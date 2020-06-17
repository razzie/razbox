package page

import (
	"fmt"

	"github.com/razzie/razlink"
)

// Welcome returns a razlink.Page that redirects the visitor to the default folder
func Welcome(defaultFolder string) *razlink.Page {
	return &razlink.Page{
		Path: "/",
		Handler: func(pr *razlink.PageRequest) *razlink.View {
			return pr.RedirectView(fmt.Sprintf("/x/%s/%s", defaultFolder, pr.Request.URL.RequestURI()))
		},
	}
}
