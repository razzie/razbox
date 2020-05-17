package razlink

import (
	"net/http"
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

// PageView is a callback function used to render the page
type PageView func(w http.ResponseWriter)

// ViewFunc is a function that produces a PageView using the input data
type ViewFunc func(data interface{}, title *string) PageView

// PageHandler handles the page's requests
// If there was no error, the handler should call viewFunc and return the resulted PageView
type PageHandler func(r *http.Request, viewFunc ViewFunc) PageView

// BindLayout creates a page renderer function that uses Razlink layout
func (page *Page) BindLayout() (func(http.ResponseWriter, *http.Request), error) {
	renderer, err := BindLayout(page.ContentTemplate, page.Stylesheets, page.Scripts, page.Meta)
	if err != nil {
		return nil, err
	}

	return func(w http.ResponseWriter, r *http.Request) {
		viewFunc := func(data interface{}, title *string) PageView {
			return func(w http.ResponseWriter) {
				if title != nil {
					renderer(w, r, *title, data)
				} else {
					renderer(w, r, page.Title, data)
				}
			}
		}

		var view PageView
		if page.Handler == nil {
			view = viewFunc(nil, nil)
		} else {
			view = page.Handler(r, viewFunc)
		}

		view(w)
	}, nil
}
