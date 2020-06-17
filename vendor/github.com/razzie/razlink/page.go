package razlink

import (
	"net/http"
	"strings"
)

// Page ...
type Page struct {
	Path            string
	Title           string
	ContentTemplate string
	Stylesheets     []string
	Scripts         []string
	Meta            map[string]string
	Handler         PageHandler
}

// PageRequest ...
type PageRequest struct {
	Request  *http.Request
	RelPath  string
	Title    string
	renderer LayoutRenderer
}

// Respond returns the default page response View
func (r *PageRequest) Respond(data interface{}, opts ...ViewOption) *View {
	v := &View{
		StatusCode: http.StatusOK,
		Data:       data,
	}
	for _, opt := range opts {
		opt(v)
	}
	v.renderer = func(w http.ResponseWriter) {
		r.renderer(w, r.Request, r.Title, data, v.StatusCode)
	}
	return v
}

// PageHandler handles the page's requests
// If there was no error, the handler should call use r.Respond(data)
type PageHandler func(r *PageRequest) *View

// BindLayout creates a http.HandlerFunc that uses Razlink layout
func (page *Page) BindLayout() (http.HandlerFunc, error) {
	renderer, err := BindLayout(page.ContentTemplate, page.Stylesheets, page.Scripts, page.Meta)
	if err != nil {
		return nil, err
	}

	return func(w http.ResponseWriter, r *http.Request) {
		pr := &PageRequest{
			Request:  r,
			RelPath:  strings.TrimPrefix(r.URL.Path, page.Path),
			Title:    page.Title,
			renderer: renderer,
		}

		var view *View
		if page.Handler != nil {
			view = page.Handler(pr)
		}
		if view == nil {
			view = pr.Respond(nil)
		}

		view.Render(w)
	}, nil
}
