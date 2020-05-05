package razbox

import (
	"fmt"
	"log"
	"net/http"
	"path"
	"path/filepath"

	"github.com/razzie/razlink"
)

func searchPageHandler(r *http.Request, view razlink.ViewFunc) razlink.PageView {
	search := r.URL.Path[8:] // skip /search/
	dir := filepath.Dir(search)
	tag := filepath.Base(search)

	folder, err := GetFolder(dir)
	if err != nil {
		log.Println(dir, "error:", err.Error())
		return razlink.ErrorView(r, "Not found", http.StatusNotFound)
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

	v := &folderPageView{
		Folder:  dir,
		Entries: entries,
	}
	title := fmt.Sprintf("%s search:%s", dir, tag)
	return view(v, &title)
}

// SearchPage ...
var SearchPage = razlink.Page{
	Path:            "/search/",
	ContentTemplate: folderPageT,
	Handler:         searchPageHandler,
}
