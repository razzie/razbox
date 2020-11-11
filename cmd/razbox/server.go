package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/razzie/beepboop"
	"github.com/razzie/razbox"
	"github.com/razzie/razbox/web/page"
)

// Server ...
type Server struct {
	srv *beepboop.Server
}

// NewServer returns a new Server
func NewServer(api *razbox.API, defaultFolder string, db *beepboop.DB) *Server {
	srv := beepboop.NewServer()
	srv.FaviconPNG, _ = ioutil.ReadFile("web/favicon.png")
	srv.AddPages(
		page.Static(),
		page.Welcome(defaultFolder),
		page.Folder(api),
		page.ReadAuth(api),
		page.WriteAuth(api),
		page.Upload(api),
		page.Download(api),
		page.Edit(api),
		page.Delete(api),
		page.Password(api),
		page.Gallery(api),
		page.Thumbnail(api),
		page.Text(api),
		page.Archive(api),
		page.CreateSubfolder(api),
		page.DeleteSubfolder(api),
	)
	srv.DB = db
	srv.Logger = log.New(os.Stdout, "", log.Lshortfile|log.LstdFlags)
	return &Server{srv: srv}
}

// Serve listens on the given port to serve http requests
func (s *Server) Serve(port int) error {
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: s.srv,
	}
	return srv.ListenAndServe()
}
