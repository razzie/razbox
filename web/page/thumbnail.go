package page

import (
	"fmt"
	"log"
	"net/http"

	"github.com/razzie/razbox/api"
	"github.com/razzie/razbox/web/page/internal"
	"github.com/razzie/razlink"
)

func thumbnailPageHandler(a *api.API, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	filename := r.URL.Path[7:] // skip /thumb/
	filename = internal.RemoveTrailingSlash(filename)

	token := a.AccessTokenFromCookies(r.Cookies())
	thumb, err := a.GetFileThumbnail(token, filename)
	if err != nil {
		//return internal.HandleError(r, err)
		switch err := err.(type) {
		case *api.ErrNoReadAccess:
			return razlink.RedirectView(r, fmt.Sprintf("/read-auth/%s?r=%s", err.Folder, r.URL.RequestURI()))
		default:
			log.Println(filename, ":", err)
		}
	}

	if thumb == nil || len(thumb.Data) == 0 {
		return razlink.RedirectView(r, "/x/"+filename)
	}

	return internal.ServeThumbnail(thumb)
}

// Thumbnail returns a razlink.Page that handles image file thumbnails
func Thumbnail(api *api.API) *razlink.Page {
	return &razlink.Page{
		Path: "/thumb/",
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return thumbnailPageHandler(api, r, view)
		},
	}
}
