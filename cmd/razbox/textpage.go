package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/razzie/razbox"
	"github.com/razzie/razlink"
)

type textPageView struct {
	Filename string
	Folder   string
	Text     string
}

var textPageT = `
<div style="clear: both">
	<div style="float: right">
		<a href="/text/{{.Folder}}/{{.Filename}}?download">&#8681; Download raw</a> |
		<a href="/x/{{.Folder}}">Go back &#10548;</a>
	</div>
</div>
<div style="max-width: 90vw">
	<pre><code>{{.Text}}</code></pre>
</div>
<script>
document.querySelectorAll('pre code').forEach((block) => {
	hljs.highlightBlock(block);
	hljs.lineNumbersBlock(block);
});
</script>
`

func textPageHandler(db *razbox.DB, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	filename := r.URL.Path[6:] // skip /text/
	filename = razbox.RemoveTrailingSlash(filename)
	dir := path.Dir(filename)

	var folder *razbox.Folder
	var err error
	folderCached := true

	if db != nil {
		folder, _ = db.GetCachedFolder(dir)
	}
	if folder == nil {
		folderCached = false
		folder, err = razbox.GetFolder(dir)
		if err != nil {
			log.Println(dir, "error:", err.Error())
			return razlink.ErrorView(r, "Folder not found", http.StatusNotFound)
		}
	}

	if db != nil && !folderCached {
		defer db.CacheFolder(folder)
	}

	hasViewAccess := folder.EnsureReadAccess(r) == nil
	basename := filepath.Base(filename)
	file, err := folder.GetFile(basename)
	if err != nil {
		if !hasViewAccess { // fake legacy behavior
			return razlink.RedirectView(r, fmt.Sprintf("/read-auth/%s?r=%s", dir, r.URL.RequestURI()))
		}
		log.Println(filename, "error:", err.Error())
		return razlink.ErrorView(r, "File not found", http.StatusNotFound)
	}

	if !file.Public && !hasViewAccess {
		return razlink.RedirectView(r, fmt.Sprintf("/read-auth/%s?r=%s", dir, r.URL.RequestURI()))
	}

	if !strings.HasPrefix(file.MIME, "text/") {
		return razlink.ErrorView(r, "Not a text file", http.StatusInternalServerError)
	}

	_, download := r.URL.Query()["download"]
	if download {
		return func(w http.ResponseWriter) {
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", basename))
			file.ServeHTTP(w, r)
		}
	}

	reader, err := file.Open()
	if err != nil {
		log.Println(filename, "error:", err.Error())
		return razlink.ErrorView(r, "Could not open file", http.StatusInternalServerError)
	}
	defer reader.Close()

	data, err := ioutil.ReadAll(reader)
	if err != nil {
		log.Println(filename, "error:", err.Error())
		return razlink.ErrorView(r, "Could not read file", http.StatusInternalServerError)
	}

	v := &textPageView{
		Filename: basename,
		Folder:   dir,
		Text:     string(data),
	}
	return view(v, &filename)
}

// GetTextPage returns a razlink.Page that visualizes text files
func GetTextPage(db *razbox.DB) *razlink.Page {
	return &razlink.Page{
		Path:            "/text/",
		ContentTemplate: textPageT,
		Stylesheets: []string{
			"/static/highlight.tomorrow.min.css",
			"/static/highlightjs-line-numbers.css",
		},
		Scripts: []string{
			"/static/highlight.min.js",
			"/static/highlightjs-line-numbers.min.js",
		},
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return textPageHandler(db, r, view)
		},
	}
}
