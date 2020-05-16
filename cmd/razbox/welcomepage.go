package main

import (
	"fmt"
	"net/http"

	"github.com/razzie/razlink"
)

// GetWelcomePage returns a razlink.Page that redirects the visitor to the default folder
func GetWelcomePage(defaultFolder string) *razlink.Page {
	return &razlink.Page{
		Path: "/",
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return razlink.RedirectView(r, fmt.Sprintf("/x/%s/%s", defaultFolder, r.URL.RequestURI()))
		},
	}
}
