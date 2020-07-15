package beepboop

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
	Metadata        map[string]string
	Handler         func(*PageRequest) *View
}

// PageRequest ...
type PageRequest struct {
	Request  *http.Request
	RelPath  string
	RelURI   string
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

// GetHandler creates a http.HandlerFunc that uses the given layout to render the page
func (page *Page) GetHandler(layout Layout) (http.HandlerFunc, error) {
	renderer, err := layout.BindTemplate(page.ContentTemplate, page.Stylesheets, page.Scripts, page.Metadata)
	if err != nil {
		return nil, err
	}

	return func(w http.ResponseWriter, r *http.Request) {
		pr := page.newPageRequest(r, renderer)

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

func (page *Page) newPageRequest(r *http.Request, renderer LayoutRenderer) *PageRequest {
	return &PageRequest{
		Request:  r,
		RelPath:  strings.TrimPrefix(r.URL.Path, page.Path),
		RelURI:   strings.TrimPrefix(r.RequestURI, page.Path),
		Title:    page.Title,
		renderer: renderer,
	}
}

func (page *Page) addMetadata(meta map[string]string) {
	if page.Metadata == nil && len(meta) > 0 {
		page.Metadata = make(map[string]string)
	}
	for name, content := range meta {
		page.Metadata[name] = content
	}
}
