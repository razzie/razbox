package page

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"path"

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

func authPageHandler(api *razbox.API, accessType string, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	dir := path.Clean(r.URL.Path[7+len(accessType):]) // skip /[accessType]-auth/
	pwPrefix := fmt.Sprintf("%s-%s", accessType, base64.StdEncoding.EncodeToString([]byte(dir)))
	v := &authPageView{
		Folder:        dir,
		PwFieldPrefix: pwPrefix,
		AccessType:    accessType,
		Redirect:      r.URL.Query().Get("r"),
	}
	if len(v.Redirect) == 0 {
		v.Redirect = "/x/" + dir
	}

	if r.Method == "POST" {
		r.ParseForm()
		pw := r.FormValue(pwPrefix + "-password")
		v.Redirect = r.FormValue("redirect")

		token, err := api.Auth(dir, accessType, pw)
		if err != nil {
			v.Error = err.Error()
			return view(v, &dir)
		}

		return razlink.CookieAndRedirectView(r, token.ToCookie(api.CookieExpiration), v.Redirect)
	}

	return view(v, &dir)
}

// ReadAuth returns a razlink.Page that handles authentication for read access
func ReadAuth(api *razbox.API) *razlink.Page {
	return &razlink.Page{
		Path:            "/read-auth/",
		ContentTemplate: GetContentTemplate("auth"),
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return authPageHandler(api, "read", r, view)
		},
	}
}

// WriteAuth returns a razlink.Page that handles authentication for read access
func WriteAuth(api *razbox.API) *razlink.Page {
	return &razlink.Page{
		Path:            "/write-auth/",
		ContentTemplate: GetContentTemplate("auth"),
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return authPageHandler(api, "write", r, view)
		},
	}
}
