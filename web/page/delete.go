package page

import (
	"path"

	"github.com/razzie/razbox"
	"github.com/razzie/razlink"
)

func deletePageHandler(api *razbox.API, pr *razlink.PageRequest) *razlink.View {
	r := pr.Request
	filename := path.Clean(pr.RelPath)
	dir := path.Dir(filename)
	redirect := r.URL.Query().Get("r")
	if len(redirect) == 0 {
		redirect = "/x/" + dir
	}

	token := api.AccessTokenFromRequest(r)
	err := api.DeleteFile(token, filename)
	if err != nil {
		return HandleError(r, err)
	}

	return pr.RedirectView(redirect)
}

// Delete returns a razlink.Page that handles deletes
func Delete(api *razbox.API) *razlink.Page {
	return &razlink.Page{
		Path: "/delete/",
		Handler: func(pr *razlink.PageRequest) *razlink.View {
			return deletePageHandler(api, pr)
		},
	}
}
