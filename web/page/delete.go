package page

import (
	"path"

	"github.com/razzie/beepboop"
	"github.com/razzie/razbox"
)

func deletePageHandler(api *razbox.API, pr *beepboop.PageRequest) *beepboop.View {
	r := pr.Request
	filename := path.Clean(pr.RelPath)
	dir := path.Dir(filename)
	redirect := r.URL.Query().Get("r")
	if len(redirect) == 0 {
		redirect = "/x/" + dir
	}

	token := beepboop.NewAccessTokenFromRequest(pr)
	err := api.DeleteFile(token, filename)
	if err != nil {
		return HandleError(r, err)
	}

	return pr.RedirectView(redirect)
}

// Delete returns a beepboop.Page that handles deletes
func Delete(api *razbox.API) *beepboop.Page {
	return &beepboop.Page{
		Path: "/delete/",
		Handler: func(pr *beepboop.PageRequest) *beepboop.View {
			return deletePageHandler(api, pr)
		},
	}
}
