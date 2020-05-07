package main

import (
	"fmt"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/razzie/razbox"
	"github.com/razzie/razlink"
)

type editPageView struct {
	Error    string
	Folder   string
	Filename string
	Tags     string
	Redirect string
}

var editPageT = `
{{if .Error}}
<strong style="color: red">{{.Error}}</strong><br /><br />
{{end}}
<form method="post">
	<input type="text" name="filename" value="{{.Filename}}" placeholder="Filename" /><br />
	<input type="text" name="tags" value="{{.Tags}}" placeholder="Tags (space separated)" /><br />
	<button>Save</button>
</form>
<div style="float: right">
	<a href="{{.Redirect}}">Go back &#10548;</a>
</div>
`

func editPageHandler(db *razbox.DB, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	filename := r.URL.Path[6:] // skip /edit/
	filename = razbox.RemoveTrailingSlash(filename)
	dir := path.Dir(filename)
	redirect := r.URL.Query().Get("r")
	if len(redirect) == 0 {
		redirect = "/x/" + dir
	}

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
			return razlink.ErrorView(r, "Folder not found", http.StatusNotFound)
		}
	}

	if db != nil && !cached {
		defer db.CacheFolder(folder)
	}

	err = folder.EnsureWriteAccess(r)
	if err != nil {
		return razlink.RedirectView(r, fmt.Sprintf("/write-auth/%s?r=%s", dir, r.URL.Path))
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
		Redirect: redirect,
	}
	title := "Edit " + filename

	if r.Method == "POST" {
		r.ParseForm()
		newName := govalidator.SafeFileName(r.FormValue("filename"))
		newTags := r.FormValue("tags")

		if newTags != v.Tags {
			file.Tags = strings.Fields(newTags)
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

// GetEditPage returns a razlink.Page that handles edits
func GetEditPage(db *razbox.DB) *razlink.Page {
	return &razlink.Page{
		Path:            "/edit/",
		ContentTemplate: editPageT,
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return editPageHandler(db, r, view)
		},
	}
}
