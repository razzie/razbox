package page

import (
	"fmt"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/razzie/razbox/lib"
	"github.com/razzie/razbox/web/page/internal"
	"github.com/razzie/razlink"
)

type editPageView struct {
	Error    string
	Folder   string
	Filename string
	Tags     string
	Public   bool
	Redirect string
	Thumb    bool
}

func editPageHandler(db *lib.DB, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	filename := r.URL.Path[6:] // skip /edit/
	filename = lib.RemoveTrailingSlash(filename)
	dir := path.Dir(filename)
	redirect := r.URL.Query().Get("r")
	if len(redirect) == 0 {
		redirect = "/x/" + dir
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

	err = folder.EnsureReadAccess(r)
	if err != nil {
		return razlink.RedirectView(r, fmt.Sprintf("/read-auth/%s?r=%s", dir, r.URL.RequestURI()))
	}

	err = folder.EnsureWriteAccess(r)
	if err != nil {
		return razlink.RedirectView(r, fmt.Sprintf("/write-auth/%s?r=%s", dir, r.URL.RequestURI()))
	}

	basename := filepath.Base(filename)
	file, err := folder.GetFile(basename)
	if err != nil {
		log.Println(filename, "error:", err.Error())
		return razlink.ErrorView(r, "File not found", http.StatusNotFound)
	}

	v := &editPageView{
		Folder:   dir,
		Filename: basename,
		Tags:     strings.Join(file.Tags, " "),
		Public:   file.Public,
		Redirect: redirect,
		Thumb:    strings.HasPrefix(file.MIME, "image/"),
	}
	title := "Edit " + filename

	if r.Method == "POST" {
		r.ParseForm()
		newName := govalidator.SafeFileName(r.FormValue("filename"))
		newTags := r.FormValue("tags")
		public := r.FormValue("public") == "public"

		if newTags != v.Tags || public != file.Public {
			file.Tags = strings.Fields(newTags)
			file.Public = public
			err := file.Save()
			if err != nil {
				v.Error = err.Error()
				return view(v, &title)
			}
		}

		if newName == "." {
			v.Error = "Invalid filename"
			return view(v, &title)
		}

		if newName != basename {
			newPath := path.Join(dir, newName)
			err := file.Move(newPath)
			if err != nil {
				v.Error = err.Error()
				return view(v, &title)
			}
		}

		folder.CachedFiles = nil
		return razlink.RedirectView(r, redirect)
	}

	return view(v, &title)
}

// Edit returns a razlink.Page that handles edits
func Edit(db *lib.DB) *razlink.Page {
	return &razlink.Page{
		Path:            "/edit/",
		ContentTemplate: internal.GetContentTemplate("edit"),
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return editPageHandler(db, r, view)
		},
	}
}
