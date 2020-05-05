package razbox

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strings"
)

// Folder ...
type Folder struct {
	Path          string `json:"-"`
	RelPath       string `json:"-"`
	Salt          string `json:"salt"`
	ReadPassword  string `json:"read_pw"`
	WritePassword string `json:"write_pw"`
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
	internalName := FilenameToUUID(basename).String()
	return GetFile(path.Join(f.Path, internalName))
}

// GetFiles ...
func (f *Folder) GetFiles() (files []*File) {
	filenames, _ := filepath.Glob(path.Join(f.Path, "????????-????-????-????-????????????.json"))
	for _, filename := range filenames {
		file, err := GetFile(filename[:len(filename)-5]) // - .json
		if err != nil {
			log.Print("GetFile error:", err)
			continue
		}
		files = append(files, file)
	}
	return
}

// GetSubfolders ...
func (f *Folder) GetSubfolders() (subfolders []string) {
	files, _ := ioutil.ReadDir(f.Path)
	for _, fi := range files {
		if fi.IsDir() {
			subfolders = append(subfolders, fi.Name())
		}
	}
	return
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

	cookie, err := r.Cookie("read:" + f.Path)
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
	cookie, err := r.Cookie("write:" + f.Path)
	if err != nil {
		return err
	}

	if cookie.Value != f.ReadPassword {
		return fmt.Errorf("incorrect write password")
	}

	return nil
}
