package razbox

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"

	"github.com/razzie/razlink"
)

func viewFile(db *DB, r *http.Request) razlink.PageView {
	filename := r.URL.Path[3:] // skip /x/
	dir := filepath.Dir(filename)

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
		return razlink.RedirectView(r, fmt.Sprintf("/read-auth/%s?r=%s", dir, r.URL.Path))
	}

	file, err := folder.GetFile(filepath.Base(filename))
	if err != nil {
		log.Println(filename, "error:", err.Error())
		return razlink.ErrorView(r, "Not found", http.StatusNotFound)
	}

	if db != nil && !cached {
		db.CacheFolder(folder)
	}

	data, err := file.Open()
	if err != nil {
		log.Println(filename, "error:", err.Error())
		return razlink.ErrorView(r, "Could not open file", http.StatusInternalServerError)
	}

	return func(w http.ResponseWriter) {
		defer data.Close()
		w.Header().Set("Content-Type", file.MIME)
		_, err := io.Copy(w, data)
		if err != nil {
			log.Println(filename, "error:", err.Error())
		}
	}
}
