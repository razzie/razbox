package beepboop

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/rs/xid"
)

// API is lightweight frontend-less version of a page
type API struct {
	Path string
	page *Page
}

// NewAPI returns a new API
func NewAPI(page *Page) *API {
	return &API{
		Path: "/api" + page.Path,
		page: page,
	}
}

// GetHandler creates a http.HandlerFunc that uses Razlink layout
func (api *API) GetHandler(ctx ContextGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pr := api.newPageRequest(r, ctx(r.Context()))
		go pr.logRequest()

		var view *View
		if api.page.Handler != nil {
			view = api.page.Handler(pr)
		}
		if view == nil {
			view = pr.Respond(nil)
		}

		renderAPIResponse(w, view)
	}
}

func (api *API) newPageRequest(r *http.Request, ctx *Context) *PageRequest {
	return &PageRequest{
		Context:   ctx,
		Request:   r,
		RequestID: xid.New().String(),
		RelPath:   strings.TrimPrefix(r.URL.Path, api.Path),
		RelURI:    strings.TrimPrefix(r.RequestURI, api.Path),
		IsAPI:     true,
	}
}

func renderAPIResponse(w http.ResponseWriter, view *View) {
	w.WriteHeader(view.StatusCode)

	if view.Error != nil {
		w.Write([]byte(view.Error.Error()))
		return
	}

	if view.Data != nil {
		data, err := json.MarshalIndent(view.Data, "", "\t")
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}
		w.Write(data)
		return
	}

	if view.StatusCode == http.StatusOK {
		w.Write([]byte("OK"))
		return
	}

	w.Write([]byte(http.StatusText(view.StatusCode)))
}
