package page

import (
	"fmt"

	"github.com/razzie/beepboop"
)

// Welcome returns a beepboop.Page that redirects the visitor to the default folder
func Welcome(defaultFolder string) *beepboop.Page {
	return &beepboop.Page{
		Path: "/",
		Handler: func(pr *beepboop.PageRequest) *beepboop.View {
			return pr.RedirectView(fmt.Sprintf("/x/%s/%s", defaultFolder, pr.Request.URL.RequestURI()))
		},
	}
}
