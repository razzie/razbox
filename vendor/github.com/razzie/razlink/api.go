package razlink

import (
	"encoding/json"
	"net/http"
	"strings"
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
func (api *API) GetHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pr := api.newPageRequest(r)

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

func (api *API) newPageRequest(r *http.Request) *PageRequest {
	return &PageRequest{
		Request: r,
		RelPath: strings.TrimPrefix(r.URL.Path, api.Path),
		RelURI:  strings.TrimPrefix(r.RequestURI, api.Path),
	}
}

func renderAPIResponse(w http.ResponseWriter, view *View) {
	w.WriteHeader(view.StatusCode)

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

	if view.Error != nil {
		w.Write([]byte(view.Error.Error()))
		return
	}

	w.Write([]byte(http.StatusText(view.StatusCode)))
}
