package razbox

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Folder ...
type Folder struct {
	Path             string   `json:"path,omitempty"`
	RelPath          string   `json:"rel_path,omitempty"`
	Salt             string   `json:"salt"`
	ReadPassword     string   `json:"read_pw"`
	WritePassword    string   `json:"write_pw"`
	CachedSubfolders []string `json:"cached_subfolders,omitempty"`
	CachedFiles      []*File  `json:"cached_files,omitempty"`
}

/*// ExploreFolders return all the folders located in the given root
func ExploreFolders(root string) (folders []*Folder, err error) {
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() && path != root {
			folder, err2 := GetFolder(filepath.Join(path, info.Name()))
			if err2 != nil {
				return err2
			}
			folders = append(folders, folder)
		}
		return nil
	})
	return
}*/

// GetFolder returns a new Folder from a handle to a .razbox file
func GetFolder(path string) (*Folder, error) {
	if len(path) > 0 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}

	if !filepath.IsAbs(path) {
		path = filepath.Join(Root, path)
	}

	if !strings.HasPrefix(path, Root) {
		return nil, fmt.Errorf("path %s is not in root (%s)", path, Root)
	}

	data, err := ioutil.ReadFile(filepath.Join(path, ".razbox"))
	if err != nil {
		return nil, err
	}

	folder := &Folder{
		Path:    path,
		RelPath: path[len(Root):],
	}
	return folder, json.Unmarshal(data, folder)
}

// GetFile ...
func (f *Folder) GetFile(basename string) (*File, error) {
	if f.CachedFiles != nil {
		for _, f := range f.CachedFiles {
			if f.Name == basename {
				return f, nil
			}
		}
	}

	internalName := FilenameToUUID(basename).String()
	filename := path.Join(f.Path, internalName)
	return GetFile(filename)
}

// GetFiles ...
func (f *Folder) GetFiles() []*File {
	if f.CachedFiles != nil {
		return f.CachedFiles
	}

	filenames, _ := filepath.Glob(path.Join(f.Path, "????????-????-????-????-????????????.json"))
	for _, filename := range filenames {
		file, err := GetFile(filename[:len(filename)-5]) // - .json
		if err != nil {
			log.Print("GetFile error:", err)
			continue
		}
		f.CachedFiles = append(f.CachedFiles, file)
	}

	sort.SliceStable(f.CachedFiles, func(i, j int) bool {
		return f.CachedFiles[i].Uploaded.Before(f.CachedFiles[j].Uploaded)
	})

	return f.CachedFiles
}

// GetSubfolders ...
func (f *Folder) GetSubfolders() []string {
	if f.CachedSubfolders != nil {
		return f.CachedSubfolders
	}

	files, _ := ioutil.ReadDir(f.Path)
	for _, fi := range files {
		if fi.IsDir() {
			f.CachedSubfolders = append(f.CachedSubfolders, fi.Name())
		}
	}
	return f.CachedSubfolders
}

// Search ...
func (f *Folder) Search(tag string) []*File {
	files := f.GetFiles()
	results := make([]*File, 0, len(files))
	for _, file := range files {
		if file.HasTag(tag) {
			results = append(results, file)
		}
	}
	return results
}

// SetPasswords ...
func (f *Folder) SetPasswords(readPw, writePw string) error {
	f.Salt = Salt()
	f.ReadPassword = Hash(f.Salt + readPw)
	f.WritePassword = Hash(f.Salt + writePw)

	if len(readPw) == 0 {
		f.ReadPassword = ""
	}

	data, _ := json.MarshalIndent(f, "", "  ")
	return ioutil.WriteFile(path.Join(f.Path, ".razbox"), data, 0644)
}

// SetReadPassword ...
func (f *Folder) SetReadPassword(readPw string) error {
	if len(readPw) == 0 {
		f.ReadPassword = ""
	} else {
		f.ReadPassword = Hash(f.Salt + readPw)
	}

	data, _ := json.MarshalIndent(f, "", "  ")
	return ioutil.WriteFile(path.Join(f.Path, ".razbox"), data, 0644)
}

// SetWritePassword ...
func (f *Folder) SetWritePassword(writePw string) error {
	f.WritePassword = Hash(f.Salt + writePw)

	data, _ := json.MarshalIndent(f, "", "  ")
	return ioutil.WriteFile(path.Join(f.Path, ".razbox"), data, 0644)
}

// EnsureReadAccess returns an error if the request doesn't contain a cookie with valid read access
func (f *Folder) EnsureReadAccess(r *http.Request) error {
	if len(f.ReadPassword) == 0 {
		return nil
	}

	cookie, err := r.Cookie("read:" + f.RelPath)
	if err != nil {
		return err
	}

	if cookie.Value != f.ReadPassword {
		return fmt.Errorf("incorrect read password")
	}

	return nil
}

// EnsureWriteAccess returns an error if the request doesn't contain a cookie with valid write access
func (f *Folder) EnsureWriteAccess(r *http.Request) error {
	cookie, err := r.Cookie("write:" + f.RelPath)
	if err != nil {
		return err
	}

	if cookie.Value != f.ReadPassword {
		return fmt.Errorf("incorrect write password")
	}

	return nil
}
