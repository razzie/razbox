package page

import (
	"fmt"
	"path"

	"github.com/razzie/beepboop"
	"github.com/razzie/razbox"
)

type galleryPageView struct {
	Folder        string                `json:"folder,omitempty"`
	Search        string                `json:"search,omitempty"`
	Entries       []*razbox.FolderEntry `json:"entries,omitempty"`
	Redirect      string                `json:"redirect,omitempty"`
	EditRedirect  string                `json:"edit_redirect,omitempty"`
	MaxThumbWidth uint                  `json:"max_thumb_width,omitempty"`
}

func galleryPageHandler(api *razbox.API, pr *beepboop.PageRequest) *beepboop.View {
	r := pr.Request
	uri := path.Clean(pr.RelPath)
	pr.Title = uri
	tag := r.URL.Query().Get("tag")
	v := &galleryPageView{
		Folder:        uri,
		Search:        tag,
		Redirect:      "/x/" + uri,
		EditRedirect:  r.URL.RequestURI(),
		MaxThumbWidth: maxThumbWidth,
	}
	if len(tag) > 0 {
		v.Redirect = fmt.Sprintf("/x/%s/?tag=%s", uri, tag)
	}

	token := beepboop.NewAccessTokenFromRequest(pr)
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

	return pr.Respond(v)
}

// Gallery returns a beepboop.Page that handles galleries
func Gallery(api *razbox.API) *beepboop.Page {
	return &beepboop.Page{
		Path:            "/gallery/",
		ContentTemplate: GetContentTemplate("gallery"),
		Stylesheets: []string{
			"/static/glightbox.min.css",
		},
		Scripts: []string{
			"/static/glightbox.min.js",
			"/static/masonry.min.js",
		},
		Handler: func(pr *beepboop.PageRequest) *beepboop.View {
			return galleryPageHandler(api, pr)
		},
	}
}
