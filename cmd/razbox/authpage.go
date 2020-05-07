package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/razzie/razbox"
	"github.com/razzie/razlink"
)

type authPageView struct {
	Error         string
	Folder        string
	PwFieldPrefix string
	AccessType    string
	Redirect      string
}

var authPageT = `
{{if .Error}}
<strong style="color: red">{{.Error}}</strong><br /><br />
{{end}}
<p>
	<strong>{{.Folder}}</strong><br />
	Enter password for <strong>{{.AccessType}}</strong> access:
</p>
<form method="post">
	<input type="password" name="{{.PwFieldPrefix}}-password" placeholder="Password" /><br />
	<input type="hidden" name="redirect" value="{{.Redirect}}" />
	<button>Enter</button>
</form>
`

func authPageHandler(db *razbox.DB, accessType string, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	uri := r.URL.Path[7+len(accessType):] // skip /[accessType]-auth/

	if len(uri) > 0 && uri[len(uri)-1] == '/' {
		uri = uri[:len(uri)-1]
	}

	pwPrefix := fmt.Sprintf("%s-%s", accessType, razbox.FilenameToUUID(uri))
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
		var folder *razbox.Folder
		var err error

		if db != nil {
			folder, _ = db.GetCachedFolder(uri)
		}
		if folder == nil {
			folder, err = razbox.GetFolder(uri)
			if err != nil {
				log.Println(uri, "error:", err.Error())
				return razlink.ErrorView(r, "Not found", http.StatusNotFound)
			}
		}

		r.ParseForm()
		pw := r.FormValue(pwPrefix + "-password")
		v.Redirect = r.FormValue("redirect")

		if folder.TestPassword(accessType, pw) {
			folder.SetPassword(accessType, pw)
			cookie := folder.GetCookie(accessType)
			return razlink.CookieAndRedirectView(r, cookie, v.Redirect)
		}

		v.Error = "Wrong password!"
	}

	return view(v, &uri)
}

// GetReadAuthPage returns a razlink.Page that handles authentication for read access
func GetReadAuthPage(db *razbox.DB) *razlink.Page {
	return &razlink.Page{
		Path:            "/read-auth/",
		ContentTemplate: authPageT,
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return authPageHandler(db, "read", r, view)
		},
	}
}

// GetWriteAuthPage returns a razlink.Page that handles authentication for read access
func GetWriteAuthPage(db *razbox.DB) *razlink.Page {
	return &razlink.Page{
		Path:            "/write-auth/",
		ContentTemplate: authPageT,
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return authPageHandler(db, "write", r, view)
		},
	}
}
