package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"path/filepath"

	"github.com/nbutton23/zxcvbn-go"
)

// Folder ...
type Folder struct {
	Root             string   `json:"root"`
	RelPath          string   `json:"rel_path"`
	Salt             string   `json:"salt"`
	ReadPassword     string   `json:"read_pw"`
	WritePassword    string   `json:"write_pw"`
	MaxFileSizeMB    int64    `json:"max_file_size"`
	CachedSubfolders []string `json:"cached_subfolders,omitempty"`
	CachedFiles      []*File  `json:"cached_files,omitempty"`
	ConfigInherited  bool     `json:"config_inherited,omitempty"`
}

// GetFolder returns a new Folder from a handle to a .razbox file
func GetFolder(root, relPath string) (*Folder, error) {
	relPath = path.Clean(relPath)
	searchPath := filepath.Join(root, relPath)
	var data []byte
	var configInherited bool
	var configFound bool

	for len(searchPath) >= len(root) {
		data, _ = ioutil.ReadFile(filepath.Join(searchPath, ".razbox"))
		if data != nil {
			configFound = true
			break
		}
		configInherited = true
		searchPath = filepath.Join(searchPath, "..")
	}

	if !configFound {
		return nil, fmt.Errorf("config file not found for folder: %s", relPath)
	}

	var folder Folder
	if len(data) > 0 {
		err := json.Unmarshal(data, &folder)
		if err != nil {
			return nil, err
		}
	}
	folder.Root = root
	folder.RelPath = relPath
	folder.ConfigInherited = configInherited

	if folder.MaxFileSizeMB < 1 {
		folder.MaxFileSizeMB = 1
	}

	return &folder, nil
}

// GetFile returns the file in the folder with the given basename
func (f *Folder) GetFile(basename string) (*File, error) {
	for _, f := range f.CachedFiles {
		if f.Name == basename {
			return f, nil
		}
	}

	internalName := FilenameToUUID(basename)
	filename := path.Join(f.RelPath, internalName)
	return getFile(f.Root, filename)
}

// CacheFile adds a file to the list of cached files
func (f *Folder) CacheFile(file *File) {
	if f.CachedFiles != nil {
		f.CachedFiles = append(f.CachedFiles, file)
	}
}

// UncacheFile removes a file from the list of cached files
func (f *Folder) UncacheFile(filename string) {
	for i, cached := range f.CachedFiles {
		if cached.Name == filename {
			//f.CachedFiles = append(f.CachedFiles[:i], f.CachedFiles[i+i:]...)
			f.CachedFiles[len(f.CachedFiles)-1], f.CachedFiles[i] = f.CachedFiles[i], f.CachedFiles[len(f.CachedFiles)-1]
			f.CachedFiles = f.CachedFiles[:len(f.CachedFiles)-1]
			return
		}
	}
}

// GetFiles returns the files in the folder
func (f *Folder) GetFiles() []*File {
	if f.CachedFiles != nil {
		return f.CachedFiles
	}

	filenames, _ := filepath.Glob(path.Join(f.Root, f.RelPath, "????????-????-????-????-????????????.json"))
	for _, filename := range filenames {
		filename = filename[len(f.Root)+1:]
		file, err := getFile(f.Root, filename[:len(filename)-5]) // - .json
		if err != nil {
			log.Print("GetFile error:", err)
			continue
		}
		f.CachedFiles = append(f.CachedFiles, file)
	}

	return f.CachedFiles
}

// GetSubfolders returns the subfolders
func (f *Folder) GetSubfolders() []string {
	if f.CachedSubfolders != nil {
		return f.CachedSubfolders
	}

	files, _ := ioutil.ReadDir(path.Join(f.Root, f.RelPath))
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
	if readPw == writePw && len(readPw) > 0 {
		return fmt.Errorf("read and write passwords cannot match")
	}
	f.Salt = Salt()
	if len(readPw) == 0 {
		f.ReadPassword = ""
	} else {
		f.ReadPassword = Hash(f.Salt + readPw)
	}
	if len(writePw) == 0 {
		f.WritePassword = ""
	} else {
		f.WritePassword = Hash(f.Salt + writePw)
	}
	return f.save()
}

// SetReadPassword sets the read password
func (f *Folder) SetReadPassword(readPw string) error {
	if len(readPw) == 0 {
		f.ReadPassword = ""
	} else {
		hash := Hash(f.Salt + readPw)
		if hash == f.WritePassword {
			return fmt.Errorf("read and write passwords cannot match")
		}
		f.ReadPassword = hash
	}
	return f.save()
}

// SetWritePassword sets the write password
func (f *Folder) SetWritePassword(writePw string) error {
	pwtest := zxcvbn.PasswordStrength(writePw, []string{f.RelPath, filepath.Base(f.RelPath)})
	if pwtest.Score < 3 {
		return fmt.Errorf("password scored too low (%d) on zxcvbn test", pwtest.Score)
	}

	hash := Hash(f.Salt + writePw)
	if hash == f.ReadPassword {
		return fmt.Errorf("read and write passwords cannot match")
	}
	f.WritePassword = hash
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
		MaxFileSizeMB: f.MaxFileSizeMB,
	}
	data, _ := json.MarshalIndent(&tmp, "", "  ")
	return ioutil.WriteFile(path.Join(f.Root, f.RelPath, ".razbox"), data, 0755)
}

// EnsureReadAccess returns an error if the access token doesn't permit read access
func (f *Folder) EnsureReadAccess(token *AccessToken) error {
	if len(f.ReadPassword) == 0 {
		return nil
	}

	pw, _ := token.Read[FilenameToUUID(f.RelPath)]
	if pw != f.ReadPassword {
		return fmt.Errorf("incorrect read password")
	}

	return nil
}

// EnsureWriteAccess returns an error if the access token doesn't permit write access
func (f *Folder) EnsureWriteAccess(token *AccessToken) error {
	if len(f.WritePassword) == 0 {
		return fmt.Errorf("folder not writable")
	}

	pw, _ := token.Write[FilenameToUUID(f.RelPath)]
	if pw != f.WritePassword {
		return fmt.Errorf("incorrect write password")
	}

	return nil
}

// EnsureAccess returns an error if the access token doesn't permit access for the given access type
func (f *Folder) EnsureAccess(accessType string, token *AccessToken) error {
	switch accessType {
	case "read":
		return f.EnsureReadAccess(token)
	case "write":
		return f.EnsureWriteAccess(token)
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
func (f *Folder) GetPasswordHash(accessType string) (string, error) {
	switch accessType {
	case "read":
		return f.ReadPassword, nil
	case "write":
		return f.WritePassword, nil
	default:
		return "", fmt.Errorf("invalid access type: %s", accessType)
	}
}

// GetAccessToken returns an access token that permits access of the given access type
func (f *Folder) GetAccessToken(accessType string) (*AccessToken, error) {
	switch accessType {
	case "read":
		pw, _ := f.GetPasswordHash(accessType)
		return &AccessToken{
			Read: map[string]string{
				FilenameToUUID(f.RelPath): pw,
			},
		}, nil

	case "write":
		pw, _ := f.GetPasswordHash(accessType)
		return &AccessToken{
			Write: map[string]string{
				FilenameToUUID(f.RelPath): pw,
			},
		}, nil

	default:
		return nil, fmt.Errorf("invalid access type: %s", accessType)
	}
}
