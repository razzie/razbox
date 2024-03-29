package page

import (
	"path"

	"github.com/razzie/beepboop"
	"github.com/razzie/razbox"
)

type folderPageView struct {
	Folder       string                `json:"folder,omitempty"`
	Search       string                `json:"search,omitempty"`
	Entries      []*razbox.FolderEntry `json:"entries,omitempty"`
	Tags         []string              `json:"tags,omitempty"`
	EditMode     bool                  `json:"edit_mode,omitempty"`
	Editable     bool                  `json:"editable,omitempty"`
	Deletable    bool                  `json:"deletable,omitempty"`
	Configurable bool                  `json:"configurable,omitempty"`
	Subfolders   bool                  `json:"subfolders,omitempty"`
	Gallery      bool                  `json:"gallery,omitempty"`
	URI          string                `json:"uri,omitempty"`
}

func folderPageHandler(api *razbox.API, pr *beepboop.PageRequest) *beepboop.View {
	r := pr.Request
	folderOrFilename := path.Clean(pr.RelPath)
	tag := r.URL.Query().Get("tag")
	entries, flags, err := api.GetFolderEntries(pr.Session(), folderOrFilename)
	if err != nil {
		return HandleError(r, err)
	}

	// this is a file
	if flags == nil {
		_, download := r.URL.Query()["download"]
		if !download {
			if entries[0].PrimaryType == "text" {
				return pr.RedirectView("/text/" + folderOrFilename)
			}
			if entries[0].Archive {
				return pr.RedirectView("/archive/" + folderOrFilename)
			}
		}

		reader, err := api.OpenFile(pr.Session(), folderOrFilename)
		if err != nil {
			return HandleError(r, err)
		}
		return pr.FileView(reader, reader.MimeType(), download)
	}

	pr.Title = folderOrFilename
	v := &folderPageView{
		Folder:       folderOrFilename,
		Search:       tag,
		EditMode:     flags.EditMode,
		Editable:     flags.Editable,
		Deletable:    flags.Deletable,
		Configurable: flags.Configurable,
		Subfolders:   flags.Subfolders,
		URI:          r.URL.RequestURI(),
	}

	for _, entry := range entries {
		if len(tag) > 0 && !entry.HasTag(tag) {
			continue
		}
		if !v.Gallery && entry.HasThumbnail {
			v.Gallery = true
		}
		v.Entries = append(v.Entries, entry)
	}
	v.Tags = collectTags(v.Entries)

	return pr.Respond(v)
}

// Folder returns a beepboop.Page that handles folders
func Folder(api *razbox.API) *beepboop.Page {
	return &beepboop.Page{
		Path:            "/x/",
		ContentTemplate: GetContentTemplate("folder"),
		Stylesheets: []string{
			"/static/glightbox.min.css",
		},
		Scripts: []string{
			"/static/glightbox.min.js",
		},
		Handler: func(pr *beepboop.PageRequest) *beepboop.View {
			return folderPageHandler(api, pr)
		},
	}
}
