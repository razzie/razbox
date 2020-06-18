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
	Error         string `json:"error,omitempty"`
	Folder        string `json:"folder,omitempty"`
	PwFieldPrefix string `json:"pw_field_prefix,omitempty"`
	AccessType    string `json:"access_type,omitempty"`
	Redirect      string `json:"redirect,omitempty"`
}

func authPageHandler(api *razbox.API, accessType string, pr *razlink.PageRequest) *razlink.View {
	r := pr.Request
	dir := path.Clean(pr.RelPath)
	pr.Title = dir
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

		token := api.AccessTokenFromRequest(r)
		newToken, err := api.Auth(token, dir, accessType, pw)
		if err != nil {
			v.Error = err.Error()
			return pr.Respond(v, razlink.WithError(err, http.StatusUnauthorized))
		}

		return pr.CookieAndRedirectView(newToken.ToCookie(api.CookieExpiration), v.Redirect)
	}

	return pr.Respond(v)
}

// ReadAuth returns a razlink.Page that handles authentication for read access
func ReadAuth(api *razbox.API) *razlink.Page {
	return &razlink.Page{
		Path:            "/read-auth/",
		ContentTemplate: GetContentTemplate("auth"),
		Handler: func(pr *razlink.PageRequest) *razlink.View {
			return authPageHandler(api, "read", pr)
		},
	}
}

// WriteAuth returns a razlink.Page that handles authentication for read access
func WriteAuth(api *razbox.API) *razlink.Page {
	return &razlink.Page{
		Path:            "/write-auth/",
		ContentTemplate: GetContentTemplate("auth"),
		Handler: func(pr *razlink.PageRequest) *razlink.View {
			return authPageHandler(api, "write", pr)
		},
	}
}
