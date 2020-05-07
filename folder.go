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

// GetFolder returns a new Folder from a handle to a .razbox file
func GetFolder(uri string) (*Folder, error) {
	if len(uri) > 0 && uri[len(uri)-1] == '/' {
		uri = uri[:len(uri)-1]
	}

	if !filepath.IsAbs(uri) {
		uri = path.Join(Root, uri)
	}

	if !strings.HasPrefix(uri, Root) {
		return nil, fmt.Errorf("path %s is not in root (%s)", uri, Root)
	}

	data, err := ioutil.ReadFile(filepath.Join(uri, ".razbox"))
	if err != nil {
		return nil, err
	}

	var relPath string
	if len(uri) > len(Root) {
		relPath = uri[len(Root)+1:]
	}

	folder := &Folder{
		Path:    uri,
		RelPath: relPath,
	}
	return folder, json.Unmarshal(data, folder)
}

// GetFile returns the file in the folder with the given basename
func (f *Folder) GetFile(basename string) (*File, error) {
	for _, f := range f.CachedFiles {
		if f.Name == basename {
			return f, nil
		}
	}

	internalName := FilenameToUUID(basename)
	filename := path.Join(f.Path, internalName)
	return GetFile(filename)
}

// GetFiles returns the files in the folder
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
		return f.CachedFiles[i].Uploaded.After(f.CachedFiles[j].Uploaded)
	})

	return f.CachedFiles
}

// GetSubfolders returns the subfolders
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

// Search returns the files that contain the given tag
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

// SetPasswords generates a random salt and sets and read and write passwords
func (f *Folder) SetPasswords(readPw, writePw string) error {
	f.Salt = Salt()
	if len(readPw) == 0 {
		f.ReadPassword = ""
	} else {
		f.ReadPassword = Hash(f.Salt + readPw)
	}
	f.WritePassword = Hash(f.Salt + writePw)
	return f.save()
}

// SetReadPassword sets the read password
func (f *Folder) SetReadPassword(readPw string) error {
	if len(readPw) == 0 {
		f.ReadPassword = ""
	} else {
		f.ReadPassword = Hash(f.Salt + readPw)
	}
	return f.save()
}

// SetWritePassword sets the write password
func (f *Folder) SetWritePassword(writePw string) error {
	f.WritePassword = Hash(f.Salt + writePw)
	return f.save()
}

// SetPassword sets the password for the given access type
func (f *Folder) SetPassword(accessType, pw string) error {
	switch accessType {
	case "read":
		return f.SetReadPassword(pw)
	case "write":
		return f.SetWritePassword(pw)
	default:
		return fmt.Errorf("invalid access type: %s", accessType)
	}
}

func (f *Folder) save() error {
	tmp := Folder{
		Salt:          f.Salt,
		ReadPassword:  f.ReadPassword,
		WritePassword: f.WritePassword,
	}
	data, _ := json.MarshalIndent(&tmp, "", "  ")
	return ioutil.WriteFile(path.Join(f.Path, ".razbox"), data, 0644)
}

// EnsureReadAccess returns an error if the request doesn't contain a cookie with valid read access
func (f *Folder) EnsureReadAccess(r *http.Request) error {
	if len(f.ReadPassword) == 0 {
		return nil
	}

	cookieName := "read-" + FilenameToUUID(f.RelPath)
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return err
	}

	if cookie.Value != f.ReadPassword {
		fmt.Println(cookie.Value, f.ReadPassword)
		return fmt.Errorf("incorrect read password")
	}

	return nil
}

// EnsureWriteAccess returns an error if the request doesn't contain a cookie with valid write access
func (f *Folder) EnsureWriteAccess(r *http.Request) error {
	cookieName := "write-" + FilenameToUUID(f.RelPath)
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return err
	}

	if cookie.Value != f.ReadPassword {
		return fmt.Errorf("incorrect write password")
	}

	return nil
}

// EnsureAccess returns an error if the request doesn't contain a valid cookie for the given access type
func (f *Folder) EnsureAccess(accessType string, r *http.Request) error {
	switch accessType {
	case "read":
		return f.EnsureReadAccess(r)
	case "write":
		return f.EnsureWriteAccess(r)
	default:
		return fmt.Errorf("invalid access type: %s", accessType)
	}
}

// TestReadPassword returns true if the given password matches the read password
func (f *Folder) TestReadPassword(readPw string) bool {
	if len(f.ReadPassword) == 0 && len(readPw) == 0 {
		return true
	}
	return Hash(f.Salt+readPw) == f.ReadPassword
}

// TestWritePassword returns true if the given password matches the write password
func (f *Folder) TestWritePassword(writePw string) bool {
	return Hash(f.Salt+writePw) == f.WritePassword
}

// TestPassword returns true if the given password matches the password for the given access type
func (f *Folder) TestPassword(accessType, pw string) bool {
	switch accessType {
	case "read":
		return f.TestReadPassword(pw)
	case "write":
		return f.TestWritePassword(pw)
	default:
		log.Print("invalid access type:", accessType)
		return false
	}
}

// GetPasswordHash returns the password hash of the given access type
func (f *Folder) GetPasswordHash(accessType string) string {
	switch accessType {
	case "read":
		return f.ReadPassword
	case "write":
		return f.WritePassword
	default:
		log.Print("invalid access type:", accessType)
		return ""
	}
}

// GetCookie returns a cookie that permits access of the given access type
func (f *Folder) GetCookie(accessType string) *http.Cookie {
	cookie := &http.Cookie{
		Name:  fmt.Sprintf("%s-%s", accessType, FilenameToUUID(f.RelPath)),
		Value: f.GetPasswordHash(accessType),
		Path:  "/",
	}
	return cookie
}
