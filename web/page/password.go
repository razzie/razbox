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

func passwordPageHandler(api *razbox.API, pr *razlink.PageRequest) *razlink.View {
	r := pr.Request
	dir := path.Clean(pr.RelPath)
	pr.Title = "Change password for " + dir
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
			return pr.Respond(v)
		}

		token := api.AccessTokenFromRequest(r)
		newToken, err := api.ChangeFolderPassword(token, dir, accessType, pw)
		if err != nil {
			v.Error = err.Error()
			return pr.Respond(v, razlink.WithError(err, http.StatusInternalServerError))
		}

		cookie := newToken.ToCookie(api.CookieExpiration)
		return pr.CookieAndRedirectView(cookie, "/x/"+dir)
	}

	token := api.AccessTokenFromRequest(r)
	flags, err := api.GetFolderFlags(token, dir)
	if err != nil {
		return HandleError(r, err)
	}

	if !flags.EditMode {
		return pr.RedirectView(
			fmt.Sprintf("/write-auth/%s?r=%s", dir, r.URL.RequestURI()),
			razlink.WithErrorMessage("Write access required", http.StatusUnauthorized))
	}

	return pr.Respond(v)
}

// Password returns a razlink.Page that handles password change for folders
func Password(api *razbox.API) *razlink.Page {
	return &razlink.Page{
		Path:            "/change-password/",
		ContentTemplate: GetContentTemplate("password"),
		Scripts: []string{
			"/static/zxcvbn.min.js",
		},
		Handler: func(pr *razlink.PageRequest) *razlink.View {
			return passwordPageHandler(api, pr)
		},
	}
}
