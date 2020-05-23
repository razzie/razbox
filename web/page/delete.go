package page

import (
	"net/http"
	"path"

	"github.com/razzie/razbox/api"
	"github.com/razzie/razbox/web/page/internal"
	"github.com/razzie/razlink"
)

func deletePageHandler(api *api.API, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	filename := r.URL.Path[8:] // skip /delete/
	filename = internal.RemoveTrailingSlash(filename)
	dir := path.Dir(filename)
	redirect := r.URL.Query().Get("r")
	if len(redirect) == 0 {
		redirect = "/x/" + dir
	}

	token := api.AccessTokenFromCookies(r.Cookies())
	err := api.DeleteFile(token, filename)
	if err != nil {
		return internal.HandleError(r, err)
	}

	return razlink.RedirectView(r, redirect)
}

// Delete returns a razlink.Page that handles deletes
func Delete(api *api.API) *razlink.Page {
	return &razlink.Page{
		Path: "/delete/",
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return deletePageHandler(api, r, view)
		},
	}
}
