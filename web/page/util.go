package page

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/razzie/beepboop"
	"github.com/razzie/razbox"
)

// GetContentTemplate returns the content template for a page
func GetContentTemplate(page string) string {
	t, err := ioutil.ReadFile(fmt.Sprintf("web/template/%s.template", page))
	if err != nil {
		panic(err)
	}
	return string(t)
}

// HandleError ...
func HandleError(r *http.Request, err error) *beepboop.View {
	switch err := err.(type) {
	case *razbox.ErrNoReadAccess:
		return beepboop.RedirectView(r,
			fmt.Sprintf("/read-auth/%s?r=%s", err.Folder, r.URL.RequestURI()),
			beepboop.WithError(err, http.StatusUnauthorized))
	case *razbox.ErrNoWriteAccess:
		return beepboop.RedirectView(r,
			fmt.Sprintf("/write-auth/%s?r=%s", err.Folder, r.URL.RequestURI()),
			beepboop.WithError(err, http.StatusUnauthorized))
	default:
		return beepboop.ErrorView(r, err.Error(), http.StatusInternalServerError)
	}
}

// ServeThumbnail ...
func ServeThumbnail(thumb *razbox.Thumbnail) *beepboop.View {
	return beepboop.HandlerView(nil, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", thumb.MIME)
		w.Header().Set("Content-Length", strconv.Itoa(len(thumb.Data)))
		w.Write(thumb.Data)
	})
}

func s(x float64) string {
	if int(x) == 1 {
		return ""
	}
	return "s"
}
