package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/razzie/razbox"
	"github.com/razzie/razlink"
)

type galleryPageView struct {
	Folder   string
	Entries  []*folderEntry
	Redirect string
}

var galleryPageT = `
<script src="https://unpkg.com/imagesloaded@4/imagesloaded.pkgd.min.js"></script>
<script src="https://unpkg.com/masonry-layout@4/dist/masonry.pkgd.min.js"></script>
<div class="grid" style="width: 90vw; max-width: 1000px">
	{{$Folder := .Folder}}
	{{range .Entries}}
		<div class="grid-item" style="width: 100%; max-width: 400px; height: auto; max-height: 400px">
			<div style="padding: 1rem">
				<a href="/x/{{.RelPath}}" target="_blank">
					<img src="/x/{{.RelPath}}" style="width: 100%; border-radius: 15px" />
				</a>
			</div>
		</div>
	{{end}}
</div>
<div style="text-align: right">
	<a href="{{.Redirect}}">Go back &#10548;</a>
</div>
<script>
imagesLoaded('.grid', function() {
	var msnry = new Masonry('.grid', {
		itemSelector: '.grid-item',
		columnWidth: 400
	});
});
</script>
`

func galleryPageHandler(db *razbox.DB, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	uri := r.URL.Path[9:] // skip /gallery/
	uri = razbox.RemoveTrailingSlash(uri)
	tag := r.URL.Query().Get("tag")

	var folder *razbox.Folder
	var err error
	cached := true

	if db != nil {
		folder, _ = db.GetCachedFolder(uri)
	}
	if folder == nil {
		cached = false
		folder, err = razbox.GetFolder(uri)
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
		return razlink.RedirectView(r, fmt.Sprintf("/read-auth/%s?r=%s", uri, r.URL.Path))
	}

	files := folder.GetFiles()
	entries := make([]*folderEntry, 0, len(files))
	for _, file := range files {
		entry := newFileEntry(uri, file)
		if !entry.IsImage {
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
		Redirect: r.URL.Query().Get("r"),
	}
	if len(v.Redirect) == 0 {
		v.Redirect = "/x/" + uri
	}

	return view(v, &uri)
}

// GetGalleryPage returns a razlink.Page that handles galleries
func GetGalleryPage(db *razbox.DB) *razlink.Page {
	return &razlink.Page{
		Path:            "/gallery/",
		ContentTemplate: galleryPageT,
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return galleryPageHandler(db, r, view)
		},
	}
}
