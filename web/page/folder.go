package page

import (
	"path"
	"strings"
	"time"

	"github.com/razzie/razbox"
	"github.com/razzie/razlink"
)

type folderPageView struct {
	Folder       string                `json:"folder,omitempty"`
	Search       string                `json:"search,omitempty"`
	Entries      []*razbox.FolderEntry `json:"entries,omitempty"`
	EditMode     bool                  `json:"edit_mode,omitempty"`
	Editable     bool                  `json:"editable,omitempty"`
	Deletable    bool                  `json:"deletable,omitempty"`
	Configurable bool                  `json:"configurable,omitempty"`
	Gallery      bool                  `json:"gallery,omitempty"`
	Redirect     string                `json:"redirect,omitempty"`
}

func folderPageHandler(api *razbox.API, pr *razlink.PageRequest) *razlink.View {
	r := pr.Request
	folderOrFilename := path.Clean(pr.RelPath)
	tag := r.URL.Query().Get("tag")
	token := api.AccessTokenFromRequest(r)
	entries, flags, err := api.GetFolderEntries(token, folderOrFilename)
	if err != nil {
		return HandleError(r, err)
	}

	// this is a file
	if len(entries) == 1 && !entries[0].Folder {
		if strings.HasPrefix(entries[0].MIME, "text/") {
			return pr.RedirectView("/text/" + folderOrFilename)
		}

		reader, err := api.OpenFile(token, folderOrFilename)
		if err != nil {
			return HandleError(r, err)
		}
		_, download := r.URL.Query()["download"]
		if download {
			return ServeFileAsAttachmentAsync(r, reader)
		}
		return ServeFileAsync(r, reader)
	}

	pr.Title = folderOrFilename
	v := &folderPageView{
		Folder:       folderOrFilename,
		Search:       tag,
		EditMode:     flags.EditMode,
		Editable:     flags.Editable,
		Deletable:    flags.Deletable,
		Configurable: flags.Configurable,
		Redirect:     r.URL.RequestURI(),
	}

	for _, entry := range entries {
		if len(tag) > 0 && !entry.HasTag(tag) {
			continue
		}
		if !v.Gallery && entry.HasThumbnail {
			v.Gallery = true
		}
		if !entry.Folder {
			entry.UploadedStr = TimeElapsed(time.Now(), time.Unix(entry.Uploaded, 0), false)
		}
		v.Entries = append(v.Entries, entry)
	}

	return pr.Respond(v)
}

// Folder returns a razlink.Page that handles folders
func Folder(api *razbox.API) *razlink.Page {
	return &razlink.Page{
		Path:            "/x/",
		ContentTemplate: GetContentTemplate("folder"),
		Handler: func(pr *razlink.PageRequest) *razlink.View {
			return folderPageHandler(api, pr)
		},
	}
}
