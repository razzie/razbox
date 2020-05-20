package razlink

import (
	"fmt"
	"net/http"
	"strconv"
)

// Server ...
type Server struct {
	mux        http.ServeMux
	FaviconPNG []byte
}

// NewServer creates a new Server
func NewServer() *Server {
	srv := &Server{
		FaviconPNG: favicon,
	}
	srv.mux.HandleFunc("/favicon.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", strconv.Itoa(len(srv.FaviconPNG)))
		_, _ = w.Write(srv.FaviconPNG)
	})
	srv.mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/favicon.png", http.StatusSeeOther)
	})
	return srv
}

// AddPage adds a new servable page to the server
func (srv *Server) AddPage(page *Page) error {
	renderer, err := page.BindLayout()
	if err != nil {
		return err
	}

	srv.mux.HandleFunc(page.Path, renderer)
	return nil
}

// AddPages adds multiple pages to the server and panics if anything goes wrong
func (srv *Server) AddPages(pages ...*Page) {
	for _, page := range pages {
		err := srv.AddPage(page)
		if err != nil {
			panic(err)
		}
	}
}

func (srv *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	srv.mux.ServeHTTP(w, r)
	fmt.Println(NewLog(r), "- path:", r.URL.Path)
}
