package page

import (
	"fmt"
	"log"
	"net/http"

	"github.com/razzie/razbox/lib"
	"github.com/razzie/razbox/web/page/internal"
	"github.com/razzie/razlink"
)

type galleryPageView struct {
	Folder   string
	Entries  []*internal.FolderEntry
	Redirect string
}

func galleryPageHandler(db *lib.DB, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	uri := r.URL.Path[9:] // skip /gallery/
	uri = lib.RemoveTrailingSlash(uri)
	tag := r.URL.Query().Get("tag")

	var folder *lib.Folder
	var err error
	cached := true

	if db != nil {
		folder, _ = db.GetCachedFolder(uri)
	}
	if folder == nil {
		cached = false
		folder, err = lib.GetFolder(uri)
		if err != nil {
			log.Println(uri, "error:", err.Error())
			return razlink.ErrorView(r, "Folder not found", http.StatusNotFound)
		}
	}

	if db != nil && !cached {
		defer db.CacheFolder(folder)
	}

	err = folder.EnsureReadAccess(r)
	if err != nil {
		return razlink.RedirectView(r, fmt.Sprintf("/read-auth/%s?r=%s", uri, r.URL.RequestURI()))
	}

	files := folder.GetFiles()
	entries := make([]*internal.FolderEntry, 0, len(files))
	for _, file := range files {
		entry := internal.NewFileEntry(uri, file)
		if !entry.HasThumbnail {
			continue
		}
		if len(tag) > 0 && !file.HasTag(tag) {
			continue
		}
		entries = append(entries, entry)
	}

	v := &galleryPageView{
		Folder:   uri,
		Entries:  entries,
		Redirect: "/x/" + uri,
	}
	if len(tag) > 0 {
		v.Redirect = fmt.Sprintf("/x/%s/?tag=%s", uri, tag)
	}

	return view(v, &uri)
}

// Gallery returns a razlink.Page that handles galleries
func Gallery(db *lib.DB) *razlink.Page {
	return &razlink.Page{
		Path:            "/gallery/",
		ContentTemplate: internal.GetContentTemplate("gallery"),
		Scripts: []string{
			"/static/masonry.min.js",
			"/static/imagesloaded.min.js",
		},
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return galleryPageHandler(db, r, view)
		},
	}
}
