package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path"
	"path/filepath"

	"github.com/razzie/razbox"
	"github.com/razzie/razlink"
)

func searchPageHandler(db *razbox.DB, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	search := r.URL.Path[8:] // skip /search/
	dir := filepath.Dir(search)
	tag := filepath.Base(search)

	var folder *razbox.Folder
	var err error
	cached := true

	if db != nil {
		folder, _ = db.GetCachedFolder(dir)
	}
	if folder == nil {
		cached = false
		folder, err = razbox.GetFolder(dir)
		if err != nil {
			log.Println(dir, "error:", err.Error())
			return razlink.ErrorView(r, "Not found", http.StatusNotFound)
		}
	}

	err = folder.EnsureReadAccess(r)
	if err != nil {
		return razlink.RedirectView(r, fmt.Sprintf("/read-auth/%s?r=%s", dir, r.URL.Path))
	}

	files := folder.Search(tag)
	entries := make([]*folderEntry, 0, len(files))

	for _, file := range files {
		entry := &folderEntry{
			Prefix:   razbox.MIMEtoSymbol(file.MIME),
			Name:     file.Name,
			RelPath:  path.Join(dir, file.Name),
			MIME:     file.MIME,
			Tags:     file.Tags,
			Size:     file.Size,
			Uploaded: file.Uploaded.Format("Mon, 02 Jan 2006 15:04:05 MST"),
		}
		entries = append(entries, entry)
	}

	if db != nil && !cached {
		db.CacheFolder(folder)
	}

	v := &folderPageView{
		Text: template.HTML(fmt.Sprintf(`
		Search results for tag: <strong>%s</strong><br />
		<a href="/x/%s">View folder content</a>
		<p></p>`, tag, dir)),
		Folder:  dir,
		Entries: entries,
	}
	title := fmt.Sprintf("%s search:%s", dir, tag)
	return view(v, &title)
}

// GetSearchPage returns a razlink.Page that handles folder search by tags
func GetSearchPage(db *razbox.DB) *razlink.Page {
	return &razlink.Page{
		Path:            "/search/",
		ContentTemplate: folderPageT,
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return searchPageHandler(db, r, view)
		},
	}
}
