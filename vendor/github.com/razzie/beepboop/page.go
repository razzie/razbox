package beepboop

import (
	"net/http"
	"os"
	"path"
	"strings"

	assetfs "github.com/elazarl/go-bindata-assetfs"
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

// GetHandler creates a http.HandlerFunc that uses the given layout to render the page
func (page *Page) GetHandler(layout Layout, ctx ContextGetter) (http.HandlerFunc, error) {
	renderer, err := layout.BindTemplate(page.ContentTemplate, page.Stylesheets, page.Scripts, page.Metadata)
	if err != nil {
		return nil, err
	}

	return func(w http.ResponseWriter, r *http.Request) {
		pr := page.newPageRequest(r, renderer, ctx(r.Context()))
		go pr.logRequest()

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

func (page *Page) newPageRequest(r *http.Request, renderer LayoutRenderer, ctx *Context) *PageRequest {
	return &PageRequest{
		Context:   ctx,
		Request:   r,
		RequestID: UniqueID(),
		RelPath:   strings.TrimPrefix(r.URL.Path, page.Path),
		RelURI:    strings.TrimPrefix(r.RequestURI, page.Path),
		IsAPI:     false,
		Title:     page.Title,
		renderer:  renderer,
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

// StaticAssetPage returns a page that serves static assets from a directory
func StaticAssetPage(pagePath, assetDir string) *Page {
	handler := func(pr *PageRequest) *View {
		uri := path.Clean(pr.RelPath)
		if fi, _ := os.Stat(path.Join(assetDir, uri)); fi != nil && fi.IsDir() {
			return pr.ErrorView("Forbidden", http.StatusForbidden)
		}
		return pr.HandlerView(func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, path.Join(assetDir, uri))
		})
	}
	return &Page{
		Path:    pagePath,
		Handler: handler,
	}
}

// AssetFSPage returns a page that serves assets from AssetFS
func AssetFSPage(pagePath string, assets *assetfs.AssetFS) *Page {
	handler := func(pr *PageRequest) *View {
		return pr.HandlerView(http.FileServer(assets).ServeHTTP)
	}
	return &Page{
		Path:    pagePath,
		Handler: handler,
	}
}
