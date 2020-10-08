package page

import (
	"fmt"
	"net/http"
	"path"

	"github.com/razzie/beepboop"
	"github.com/razzie/razbox"
)

const maxThumbWidth = 250

func thumbnailPageHandler(api *razbox.API, pr *beepboop.PageRequest) *beepboop.View {
	r := pr.Request
	filename := path.Clean(pr.RelPath)
	token := beepboop.NewAccessTokenFromRequest(pr)
	thumb, err := api.GetFileThumbnail(token, filename, maxThumbWidth)
	if err != nil {
		switch err := err.(type) {
		case *razbox.ErrNoReadAccess:
			return pr.RedirectView(
				fmt.Sprintf("/read-auth/%s?r=%s", err.Folder, r.URL.RequestURI()),
				beepboop.WithErrorMessage("Read access required", http.StatusUnauthorized))
		default:
			pr.Log(filename, ":", err)
		}
	}

	if thumb == nil || len(thumb.Data) == 0 {
		return pr.RedirectView("/x/" + filename)
	}

	return ServeThumbnail(thumb)
}

// Thumbnail returns a beepboop.Page that handles image file thumbnails
func Thumbnail(api *razbox.API) *beepboop.Page {
	return &beepboop.Page{
		Path: "/thumb/",
		Handler: func(pr *beepboop.PageRequest) *beepboop.View {
			return thumbnailPageHandler(api, pr)
		},
	}
}
