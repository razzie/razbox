package page

import (
	"fmt"
	"log"
	"net/http"

	"github.com/razzie/razbox/lib"
	"github.com/razzie/razbox/web/page/internal"
	"github.com/razzie/razlink"
)

type passwordPageView struct {
	Error         string
	Folder        string
	PwFieldPrefix string
	WriteAccess   bool
}

func passwordPageHandler(db *lib.DB, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	uri := r.URL.Path[17:] // skip /change-password/
	uri = lib.RemoveTrailingSlash(uri)

	var folder *lib.Folder
	var err error
	cached := true

	if db != nil {
		folder, _ = db.GetCachedFolder(uri)
	}
	if folder == nil {
		cached = false
		folder, err = lib.GetFolder(uri)
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
		return razlink.RedirectView(r, fmt.Sprintf("/read-auth/%s?r=%s", uri, r.URL.RequestURI()))
	}

	err = folder.EnsureWriteAccess(r)
	if err != nil {
		return razlink.RedirectView(r, fmt.Sprintf("/write-auth/%s?r=%s", uri, r.URL.RequestURI()))
	}

	title := "Change password for " + uri
	v := passwordPageView{
		Folder:        uri,
		PwFieldPrefix: lib.FilenameToUUID(uri),
	}

	if r.Method == "POST" {
		r.ParseForm()
		accessType := r.FormValue("access_type")
		pw := r.FormValue(v.PwFieldPrefix + "-password")
		pwconfirm := r.FormValue(v.PwFieldPrefix + "-password-confirm")

		if accessType == "write" {
			v.WriteAccess = true
		}

		if folder.ConfigInherited {
			v.Error = "Cannot change password of folders that inherit parent configuration"
			return view(v, &title)
		}

		if pw != pwconfirm {
			v.Error = "Password mismatch"
			return view(v, &title)
		}

		err := folder.SetPassword(accessType, pw)
		if err != nil {
			v.Error = err.Error()
			return view(v, &title)
		}

		cookie := folder.GetCookie(accessType)
		return razlink.CookieAndRedirectView(r, cookie, "/x/"+uri)
	}

	return view(v, &title)
}

// Password returns a razlink.Page that handles password change for folders
func Password(db *lib.DB) *razlink.Page {
	return &razlink.Page{
		Path:            "/change-password/",
		ContentTemplate: internal.GetContentTemplate("password"),
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return passwordPageHandler(db, r, view)
		},
	}
}
