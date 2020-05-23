package page

import (
	"fmt"
	"net/http"

	"github.com/razzie/razbox"
	"github.com/razzie/razbox/internal"
	"github.com/razzie/razlink"
)

type galleryPageView struct {
	Folder   string
	Entries  []*razbox.FolderEntry
	Redirect string
}

func galleryPageHandler(api *razbox.API, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	uri := r.URL.Path[9:] // skip /gallery/
	uri = internal.RemoveTrailingSlash(uri)
	tag := r.URL.Query().Get("tag")

	v := &galleryPageView{
		Folder:   uri,
		Redirect: "/x/" + uri,
	}
	if len(tag) > 0 {
		v.Redirect = fmt.Sprintf("/x/%s/?tag=%s", uri, tag)
	}

	token := api.AccessTokenFromCookies(r.Cookies())
	entries, _, err := api.GetFolderEntries(token, uri)
	if err != nil {
		return HandleError(r, err)
	}

	for _, entry := range entries {
		if !entry.HasThumbnail {
			continue
		}
		if len(tag) > 0 && !entry.HasTag(tag) {
			continue
		}
		v.Entries = append(v.Entries, entry)
	}

	return view(v, &uri)
}

// Gallery returns a razlink.Page that handles galleries
func Gallery(api *razbox.API) *razlink.Page {
	return &razlink.Page{
		Path:            "/gallery/",
		ContentTemplate: GetContentTemplate("gallery"),
		Scripts: []string{
			"/static/masonry.min.js",
			"/static/imagesloaded.min.js",
		},
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return galleryPageHandler(api, r, view)
		},
	}
}