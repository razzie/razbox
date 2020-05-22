package page

import (
	"fmt"
	"log"
	"net/http"

	"github.com/razzie/razbox/lib"
	"github.com/razzie/razbox/web/page/internal"
	"github.com/razzie/razlink"
)

type authPageView struct {
	Error         string
	Folder        string
	PwFieldPrefix string
	AccessType    string
	Redirect      string
}

func authPageHandler(db *lib.DB, accessType string, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	uri := r.URL.Path[7+len(accessType):] // skip /[accessType]-auth/
	uri = lib.RemoveTrailingSlash(uri)

	pwPrefix := fmt.Sprintf("%s-%s", accessType, lib.FilenameToUUID(uri))
	v := &authPageView{
		Folder:        uri,
		PwFieldPrefix: pwPrefix,
		AccessType:    accessType,
		Redirect:      r.URL.Query().Get("r"),
	}
	if len(v.Redirect) == 0 {
		v.Redirect = "/x/" + uri
	}

	if r.Method == "POST" {
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

		r.ParseForm()
		pw := r.FormValue(pwPrefix + "-password")
		v.Redirect = r.FormValue("redirect")

		if folder.TestPassword(accessType, pw) {
			cookie := folder.GetCookie(accessType)
			return razlink.CookieAndRedirectView(r, cookie, v.Redirect)
		}

		v.Error = "Wrong password!"
	}

	return view(v, &uri)
}

// ReadAuth returns a razlink.Page that handles authentication for read access
func ReadAuth(db *lib.DB) *razlink.Page {
	return &razlink.Page{
		Path:            "/read-auth/",
		ContentTemplate: internal.GetContentTemplate("auth"),
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return authPageHandler(db, "read", r, view)
		},
	}
}

// WriteAuth returns a razlink.Page that handles authentication for read access
func WriteAuth(db *lib.DB) *razlink.Page {
	return &razlink.Page{
		Path:            "/write-auth/",
		ContentTemplate: internal.GetContentTemplate("auth"),
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return authPageHandler(db, "write", r, view)
		},
	}
}
