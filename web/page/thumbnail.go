package page

import (
	"fmt"
	"log"
	"net/http"
	"path"

	"github.com/razzie/razbox"
	"github.com/razzie/razlink"
)

const maxThumbWidth = 250

func thumbnailPageHandler(api *razbox.API, pr *razlink.PageRequest) *razlink.View {
	r := pr.Request
	filename := path.Clean(pr.RelPath)
	token := api.AccessTokenFromRequest(r)
	thumb, err := api.GetFileThumbnail(token, filename, maxThumbWidth)
	if err != nil {
		switch err := err.(type) {
		case *razbox.ErrNoReadAccess:
			return pr.RedirectView(
				fmt.Sprintf("/read-auth/%s?r=%s", err.Folder, r.URL.RequestURI()),
				razlink.WithErrorMessage("Read access required", http.StatusUnauthorized))
		default:
			log.Println(filename, ":", err)
		}
	}

	if thumb == nil || len(thumb.Data) == 0 {
		return pr.RedirectView("/x/" + filename)
	}

	return ServeThumbnail(thumb)
}

// Thumbnail returns a razlink.Page that handles image file thumbnails
func Thumbnail(api *razbox.API) *razlink.Page {
	return &razlink.Page{
		Path: "/thumb/",
		Handler: func(pr *razlink.PageRequest) *razlink.View {
			return thumbnailPageHandler(api, pr)
		},
	}
}
