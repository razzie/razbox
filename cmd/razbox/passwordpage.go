package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/razzie/razbox"
	"github.com/razzie/razlink"
)

type passwordPageView struct {
	Error         string
	Folder        string
	PwFieldPrefix string
	WriteAccess   bool
}

var passwordPageT = `
{{if .Error}}
<strong style="color: red">{{.Error}}</strong><br /><br />
{{end}}
<form method="post">
	&#128273; Change password for <select name="access_type">
		{{if .WriteAccess}}
			<option value="read">read</option>
			<option value="write" selected>write</option>
		{{else}}
			<option value="read" selected>read</option>
			<option value="write">write</option>
		{{end}}
	</select> access:
	<p>
		<input type="password" name="{{.PwFieldPrefix}}-password" placeholder="Password" /><br />
		<input type="password" name="{{.PwFieldPrefix}}-password-confirm" placeholder="Password confirm" /><br />
		<div style="clear: both">
			<button>Save</button>
			<a href="/x/{{.Folder}}" style="float: right">Go back &#10548;</a>
		</div>
	</p>
</form>
<div>
	<small>read password can be empty to allow public access</small><br />
	<small>write password must score at least 3/4 on <a href="https://lowe.github.io/tryzxcvbn/">zxcvbn</a> test</small>
</div>
`

func passwordPageHandler(db *razbox.DB, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	uri := r.URL.Path[17:] // skip /change-password/
	uri = razbox.RemoveTrailingSlash(uri)

	var folder *razbox.Folder
	var err error
	cached := true

	if db != nil {
		folder, _ = db.GetCachedFolder(uri)
	}
	if folder == nil {
		cached = false
		folder, err = razbox.GetFolder(uri)
		if err != nil {
			log.Println(uri, "error:", err.Error())
			return razlink.ErrorView(r, "Folder not found", http.StatusNotFound)
		}
	}

	if db != nil && !cached {
		defer db.CacheFolder(folder)
	}

	err = folder.EnsureWriteAccess(r)
	if err != nil {
		return razlink.RedirectView(r, fmt.Sprintf("/write-auth/%s?r=%s", uri, r.URL.RequestURI()))
	}

	title := "Change password for " + uri
	v := passwordPageView{
		Folder:        uri,
		PwFieldPrefix: razbox.FilenameToUUID(uri),
	}

	if r.Method == "POST" {
		r.ParseForm()
		accessType := r.FormValue("access_type")
		pw := r.FormValue(v.PwFieldPrefix + "-password")
		pwconfirm := r.FormValue(v.PwFieldPrefix + "-password-confirm")

		if accessType == "write" {
			v.WriteAccess = true
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

// GetPasswordPage returns a razlink.Page that handles password change for folders
func GetPasswordPage(db *razbox.DB) *razlink.Page {
	return &razlink.Page{
		Path:            "/change-password/",
		ContentTemplate: passwordPageT,
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return passwordPageHandler(db, r, view)
		},
	}
}
