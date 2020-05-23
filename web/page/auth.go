package page

import (
	"fmt"
	"net/http"

	"github.com/razzie/razbox"
	"github.com/razzie/razbox/internal"
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
	uri := r.URL.Path[7+len(accessType):] // skip /[accessType]-auth/
	uri = internal.RemoveTrailingSlash(uri)

	pwPrefix := fmt.Sprintf("%s-%s", accessType, internal.FilenameToUUID(uri))
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
		r.ParseForm()
		pw := r.FormValue(pwPrefix + "-password")
		v.Redirect = r.FormValue("redirect")

		a, err := api.Auth(uri, accessType, pw)
		if err != nil {
			v.Error = err.Error()
			return view(v, &uri)
		}

		return razlink.CookieAndRedirectView(r, a.ToCookie(), v.Redirect)
	}

	return view(v, &uri)
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
