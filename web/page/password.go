package page

import (
	"net/http"

	"github.com/razzie/razbox"
	"github.com/razzie/razbox/internal"
	"github.com/razzie/razlink"
)

type passwordPageView struct {
	Error         string
	Folder        string
	PwFieldPrefix string
	WriteAccess   bool
}

func passwordPageHandler(api *razbox.API, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	uri := r.URL.Path[17:] // skip /change-password/
	uri = internal.RemoveTrailingSlash(uri)

	title := "Change password for " + uri
	v := passwordPageView{
		Folder:        uri,
		PwFieldPrefix: internal.FilenameToUUID(uri),
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

		token := api.AccessTokenFromCookies(r.Cookies())
		newToken, err := api.ChangeFolderPassword(token, uri, accessType, pw)
		if err != nil {
			v.Error = err.Error()
			return view(v, &title)
		}

		cookie := newToken.ToCookie()
		return razlink.CookieAndRedirectView(r, cookie, "/x/"+uri)
	}

	return view(v, &title)
}

// Password returns a razlink.Page that handles password change for folders
func Password(api *razbox.API) *razlink.Page {
	return &razlink.Page{
		Path:            "/change-password/",
		ContentTemplate: GetContentTemplate("password"),
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return passwordPageHandler(api, r, view)
		},
	}
}