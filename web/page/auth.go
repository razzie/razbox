package page

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"path"

	"github.com/razzie/beepboop"
	"github.com/razzie/razbox"
)

type authPageView struct {
	Error         string `json:"error,omitempty"`
	Folder        string `json:"folder,omitempty"`
	PwFieldPrefix string `json:"pw_field_prefix,omitempty"`
	AccessType    string `json:"access_type,omitempty"`
	Redirect      string `json:"redirect,omitempty"`
}

func authPageHandler(api *razbox.API, accessType string, pr *beepboop.PageRequest) *beepboop.View {
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

		if err := api.Auth(pr, dir, accessType, pw); err != nil {
			v.Error = err.Error()
			return pr.Respond(v, beepboop.WithError(err, http.StatusUnauthorized))
		}

		return pr.RedirectView(v.Redirect)
	}

	return pr.Respond(v)
}

// ReadAuth returns a beepboop.Page that handles authentication for read access
func ReadAuth(api *razbox.API) *beepboop.Page {
	return &beepboop.Page{
		Path:            "/read-auth/",
		ContentTemplate: GetContentTemplate("auth"),
		Handler: func(pr *beepboop.PageRequest) *beepboop.View {
			return authPageHandler(api, "read", pr)
		},
	}
}

// WriteAuth returns a beepboop.Page that handles authentication for read access
func WriteAuth(api *razbox.API) *beepboop.Page {
	return &beepboop.Page{
		Path:            "/write-auth/",
		ContentTemplate: GetContentTemplate("auth"),
		Handler: func(pr *beepboop.PageRequest) *beepboop.View {
			return authPageHandler(api, "write", pr)
		},
	}
}
