package razbox

import (
	"io"
	"log"
	"net/http"
	"path/filepath"

	"github.com/razzie/razlink"
)

func viewFile(r *http.Request) razlink.PageView {
	filename := r.URL.Path[3:] // skip /x/

	folder, err := GetFolder(filepath.Dir(filename))
	if err != nil {
		log.Println(filename, "error:", err.Error())
		return razlink.ErrorView(r, "Not found", http.StatusNotFound)
	}

	err = folder.EnsureReadAccess(r)
	if err != nil {
		log.Println(filename, "error:", err.Error())
		return razlink.ErrorView(r, "Unauthorized", http.StatusUnauthorized)
	}

	file, err := folder.GetFile(filepath.Base(filename))
	if err != nil {
		log.Println(filename, "error:", err.Error())
		return razlink.ErrorView(r, "Not found", http.StatusNotFound)
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
