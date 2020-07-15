package main

import (
	"io/ioutil"

	"github.com/razzie/beepboop"
	"github.com/razzie/razbox"
	"github.com/razzie/razbox/web/page"
)

// NewServer ...
func NewServer(api *razbox.API, defaultFolder string) *beepboop.Server {
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
		page.CreateSubfolder(api),
		page.DeleteSubfolder(api),
	)
	return srv
}
