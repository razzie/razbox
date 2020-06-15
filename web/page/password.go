package page

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"path"

	"github.com/razzie/razbox"
	"github.com/razzie/razlink"
)

type passwordPageView struct {
	Error         string
	Folder        string
	PwFieldPrefix string
	WriteAccess   bool
}

func passwordPageHandler(api *razbox.API, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	dir := path.Clean(r.URL.Path[17:]) // skip /change-password/
	title := "Change password for " + dir
	v := passwordPageView{
		Folder:        dir,
		PwFieldPrefix: base64.StdEncoding.EncodeToString([]byte(dir)),
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

		token := api.AccessTokenFromRequest(r)
		newToken, err := api.ChangeFolderPassword(token, dir, accessType, pw)
		if err != nil {
			v.Error = err.Error()
			return view(v, &title)
		}

		cookie := newToken.ToCookie(api.CookieExpiration)
		return razlink.CookieAndRedirectView(r, cookie, "/x/"+dir)
	}

	token := api.AccessTokenFromRequest(r)
	flags, err := api.GetFolderFlags(token, dir)
	if err != nil {
		return HandleError(r, err)
	}

	if !flags.EditMode {
		return razlink.RedirectView(r, fmt.Sprintf("/write-auth/%s?r=%s", dir, r.URL.RequestURI()))
	}

	return view(v, &title)
}

// Password returns a razlink.Page that handles password change for folders
func Password(api *razbox.API) *razlink.Page {
	return &razlink.Page{
		Path:            "/change-password/",
		ContentTemplate: GetContentTemplate("password"),
		Scripts: []string{
			"/static/zxcvbn.min.js",
		},
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return passwordPageHandler(api, r, view)
		},
	}
}
