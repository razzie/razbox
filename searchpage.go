package razbox

import (
	"fmt"
	"log"
	"net/http"
	"path"
	"path/filepath"

	"github.com/razzie/razlink"
)

func searchPageHandler(db *DB, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	search := r.URL.Path[8:] // skip /search/
	dir := filepath.Dir(search)
	tag := filepath.Base(search)

	var folder *Folder
	var err error
	cached := true

	if db != nil {
		folder, _ = db.GetCachedFolder(dir)
	}
	if folder == nil {
		cached = false
		folder, err = GetFolder(dir)
		if err != nil {
			log.Println(dir, "error:", err.Error())
			return razlink.ErrorView(r, "Not found", http.StatusNotFound)
		}
	}

	err = folder.EnsureReadAccess(r)
	if err != nil {
		log.Println(dir, "error:", err.Error())
		return razlink.ErrorView(r, "Unauthorized", http.StatusUnauthorized)
	}

	files := folder.Search(tag)
	entries := make([]*folderEntry, 0, len(files))

	for _, file := range files {
		entry := &folderEntry{
			Name:     file.Name,
			RelPath:  path.Join(dir, file.Name),
			MIME:     file.MIME,
			Tags:     file.Tags,
			Size:     file.Size,
			Uploaded: file.Uploaded,
		}
		entries = append(entries, entry)
	}

	if db != nil && !cached {
		db.CacheFolder(folder)
	}

	v := &folderPageView{
		Folder:  dir,
		Entries: entries,
	}
	title := fmt.Sprintf("%s search:%s", dir, tag)
	return view(v, &title)
}

// GetSearchPage returns a razlink.Page that handles folder search by tags
func GetSearchPage(db *DB) *razlink.Page {
	return &razlink.Page{
		Path:            "/search/",
		ContentTemplate: folderPageT,
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return searchPageHandler(db, r, view)
		},
	}
}
