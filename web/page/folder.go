package page

import (
	"fmt"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/razzie/razbox/lib"
	"github.com/razzie/razbox/web/page/internal"
	"github.com/razzie/razlink"
)

type folderPageView struct {
	Folder       string
	Search       string
	Entries      []*internal.FolderEntry
	EditMode     bool
	Editable     bool
	Configurable bool
	Gallery      bool
	Redirect     string
	SortRedirect string
}

func folderPageHandler(db *lib.DB, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	uri := r.URL.Path[3:] // skip /x/
	uri = lib.RemoveTrailingSlash(uri)
	tag := r.URL.Query().Get("tag")
	sortOrder := r.URL.Query().Get("sort")

	var filename string
	dir := uri
	if !lib.IsFolder(uri) {
		dir = path.Dir(uri)
		filename = uri
	}

	var folder *lib.Folder
	var err error
	cached := true

	if db != nil {
		folder, _ = db.GetCachedFolder(dir)
	}
	if folder == nil {
		cached = false
		folder, err = lib.GetFolder(dir)
		if err != nil {
			log.Println(dir, "error:", err.Error())
			return razlink.ErrorView(r, "Folder not found", http.StatusNotFound)
		}
	}

	if db != nil && !cached {
		defer db.CacheFolder(folder)
	}

	hasViewAccess := folder.EnsureReadAccess(r) == nil

	if len(filename) > 0 {
		file, err := folder.GetFile(filepath.Base(filename))
		if err != nil {
			if !hasViewAccess { // fake legacy behavior
				return razlink.RedirectView(r, fmt.Sprintf("/read-auth/%s?r=%s", dir, r.URL.RequestURI()))
			}
			log.Println(filename, "error:", err.Error())
			return razlink.ErrorView(r, "File not found", http.StatusNotFound)
		}

		if hasViewAccess || file.Public {
			if strings.HasPrefix(file.MIME, "text/") {
				return razlink.RedirectView(r, "/text/"+filename)
			}
			return func(w http.ResponseWriter) {
				file.ServeHTTP(w, r)
			}
		}
	}

	if !hasViewAccess {
		return razlink.RedirectView(r, fmt.Sprintf("/read-auth/%s?r=%s", dir, r.URL.RequestURI()))
	}

	v := &folderPageView{
		Folder:       uri,
		Search:       tag,
		EditMode:     folder.EnsureWriteAccess(r) == nil,
		Editable:     len(folder.WritePassword) > 0,
		Configurable: !folder.ConfigInherited,
		Redirect:     r.URL.RequestURI(),
		SortRedirect: r.URL.Path + "?tag=" + tag,
	}

	if len(tag) == 0 {
		subfolders := folder.GetSubfolders()
		if len(uri) > 0 {
			entry := internal.NewSubfolderEntry(uri, "..")
			v.Entries = append(v.Entries, entry)
		}
		for _, subfolder := range subfolders {
			entry := internal.NewSubfolderEntry(uri, subfolder)
			v.Entries = append(v.Entries, entry)
		}
	}

	for _, file := range folder.GetFiles() {
		if len(tag) > 0 && !file.HasTag(tag) {
			continue
		}

		entry := internal.NewFileEntry(uri, file)
		entry.EditMode = v.EditMode
		if !v.Gallery && entry.HasThumbnail {
			v.Gallery = true
		}
		v.Entries = append(v.Entries, entry)
	}

	internal.SortFolderEntries(v.Entries, sortOrder)
	return view(v, &uri)
}

// Folder returns a razlink.Page that handles folders
func Folder(db *lib.DB) *razlink.Page {
	return &razlink.Page{
		Path:            "/x/",
		ContentTemplate: internal.GetContentTemplate("folder"),
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return folderPageHandler(db, r, view)
		},
	}
}
