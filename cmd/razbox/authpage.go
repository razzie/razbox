package main

import (
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

func readAuthPageHandler(r *http.Request, view razlink.ViewFunc) razlink.PageView {
	uri := r.URL.Path[11:] // skip /read-auth/

	if len(uri) > 0 && uri[len(uri)-1] == '/' {
		uri = uri[:len(uri)-1]
	}

	pwPrefix := "read-" + razbox.FilenameToUUID(uri)
	v := &authPageView{
		Folder:        uri,
		PwFieldPrefix: pwPrefix,
		AccessType:    "read",
		Redirect:      r.URL.Query().Get("r"),
	}
	if len(v.Redirect) == 0 {
		v.Redirect = "/x/" + uri
	}

	if r.Method == "POST" {
		folder, err := razbox.GetFolder(uri)
		if err != nil {
			log.Println(uri, "error:", err.Error())
			return razlink.ErrorView(r, "Not found", http.StatusNotFound)
		}

		r.ParseForm()
		pw := r.FormValue(pwPrefix + "-password")
		v.Redirect = r.FormValue("redirect")

		if folder.TestReadPassword(pw) {
			folder.SetReadPassword(pw)
			cookie := &http.Cookie{
				Name:  pwPrefix,
				Value: folder.ReadPassword,
				Path:  "/",
			}
			return razlink.CookieAndRedirectView(r, cookie, v.Redirect)
		}

		v.Error = "Wrong password!"
	}

	return view(v, &uri)
}

func writeAuthPageHandler(r *http.Request, view razlink.ViewFunc) razlink.PageView {
	uri := r.URL.Path[12:] // skip /write-auth/

	if len(uri) > 0 && uri[len(uri)-1] == '/' {
		uri = uri[:len(uri)-1]
	}

	pwPrefix := "write-" + razbox.FilenameToUUID(uri)
	v := &authPageView{
		Folder:        uri,
		PwFieldPrefix: pwPrefix,
		AccessType:    "write",
		Redirect:      r.Referer(),
	}
	if len(v.Redirect) == 0 {
		v.Redirect = "/x/" + uri
	}

	if r.Method == "POST" {
		folder, err := razbox.GetFolder(uri)
		if err != nil {
			log.Println(uri, "error:", err.Error())
			return razlink.ErrorView(r, "Not found", http.StatusNotFound)
		}

		r.ParseForm()
		pw := r.FormValue(pwPrefix + "-password")
		v.Redirect = r.FormValue("redirect")

		if folder.TestWritePassword(pw) {
			folder.SetWritePassword(pw)
			cookie := &http.Cookie{
				Name:  pwPrefix,
				Value: folder.WritePassword,
				Path:  "/",
			}
			return razlink.CookieAndRedirectView(r, cookie, v.Redirect)
		}

		v.Error = "Wrong password!"
	}

	return view(v, &uri)
}

// ReadAuthPage handles authentication for read access
var ReadAuthPage = razlink.Page{
	Path:            "/read-auth/",
	ContentTemplate: authPageT,
	Handler:         readAuthPageHandler,
}

// WriteAuthPage handles authentication for write access
var WriteAuthPage = razlink.Page{
	Path:            "/write-auth/",
	ContentTemplate: authPageT,
	Handler:         writeAuthPageHandler,
}
