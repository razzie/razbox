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

func thumbnailPageHandler(api *razbox.API, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	filename := path.Clean(r.URL.Path[7:]) // skip /thumb/
	token := api.AccessTokenFromCookies(r.Cookies())
	thumb, err := api.GetFileThumbnail(token, filename, maxThumbWidth)
	if err != nil {
		switch err := err.(type) {
		case *razbox.ErrNoReadAccess:
			return razlink.RedirectView(r, fmt.Sprintf("/read-auth/%s?r=%s", err.Folder, r.URL.RequestURI()))
		default:
			log.Println(filename, ":", err)
		}
	}

	if thumb == nil || len(thumb.Data) == 0 {
		return razlink.RedirectView(r, "/x/"+filename)
	}

	return ServeThumbnail(thumb)
}

// Thumbnail returns a razlink.Page that handles image file thumbnails
func Thumbnail(api *razbox.API) *razlink.Page {
	return &razlink.Page{
		Path: "/thumb/",
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return thumbnailPageHandler(api, r, view)
		},
	}
}
