package main

import (
	"io/ioutil"

	"github.com/razzie/razbox"
	"github.com/razzie/razbox/web/page"
	"github.com/razzie/razlink"
)

// NewServer ...
func NewServer(api *razbox.API, defaultFolder string) *razlink.Server {
	srv := razlink.NewServer()
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
	)
	return srv
}
