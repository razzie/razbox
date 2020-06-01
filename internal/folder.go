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

// FolderConfig stores the folder's passwords and other congfiguration
type FolderConfig struct {
	Salt          string `json:"salt"`
	ReadPassword  string `json:"read_pw"`
	WritePassword string `json:"write_pw"`
	MaxFileSizeMB int64  `json:"max_file_size"`
}

// Folder ...
type Folder struct {
	Root             string       `json:"root"`
	RelPath          string       `json:"rel_path"`
	Config           FolderConfig `json:"config"`
	ConfigInherited  bool         `json:"config_inherited"`
	ConfigRootFolder string       `json:"config_root"`
	CachedSubfolders []string     `json:"cached_subfolders"`
	CachedFiles      []*File      `json:"cached_files"`
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

	folder := &Folder{
		Root:             root,
		RelPath:          relPath,
		ConfigInherited:  configInherited,
		ConfigRootFolder: searchPath[len(root):],
	}

	if len(data) > 0 {
		err := json.Unmarshal(data, &folder.Config)
		if err != nil {
			return nil, err
		}
	}

	if folder.Config.MaxFileSizeMB < 1 {
		folder.Config.MaxFileSizeMB = 1
	}

	return folder, nil
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

// CacheSubfolder adds a subfolder to the list of cached subfolders
func (f *Folder) CacheSubfolder(subfolder string) {
	if f.CachedSubfolders != nil {
		f.CachedSubfolders = append(f.CachedSubfolders, subfolder)
	}
}

// UncacheSubfolder removes a subfolder from the list of cached subfolders
func (f *Folder) UncacheSubfolder(subfolder string) {
	for i, cached := range f.CachedSubfolders {
		if cached == subfolder {
			//f.CachedSubfolders = append(f.CachedSubfolders[:i], f.CachedSubfolders[i+i:]...)
			f.CachedSubfolders[len(f.CachedSubfolders)-1], f.CachedSubfolders[i] =
				f.CachedSubfolders[i], f.CachedSubfolders[len(f.CachedSubfolders)-1]
			f.CachedSubfolders = f.CachedSubfolders[:len(f.CachedSubfolders)-1]
			return
		}
	}
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
	f.Config.Salt = Salt()
	if len(readPw) == 0 {
		f.Config.ReadPassword = ""
	} else {
		f.Config.ReadPassword = Hash(f.Config.Salt + readPw)
	}
	if len(writePw) == 0 {
		f.Config.WritePassword = ""
	} else {
		f.Config.WritePassword = Hash(f.Config.Salt + writePw)
	}
	return f.save()
}

// SetReadPassword sets the read password
func (f *Folder) SetReadPassword(readPw string) error {
	if len(readPw) == 0 {
		f.Config.ReadPassword = ""
	} else {
		hash := Hash(f.Config.Salt + readPw)
		if hash == f.Config.WritePassword {
			return fmt.Errorf("read and write passwords cannot match")
		}
		f.Config.ReadPassword = hash
	}
	return f.save()
}

// SetWritePassword sets the write password
func (f *Folder) SetWritePassword(writePw string) error {
	pwtest := zxcvbn.PasswordStrength(writePw, []string{f.RelPath, filepath.Base(f.RelPath)})
	if pwtest.Score < 3 {
		return fmt.Errorf("password scored too low (%d) on zxcvbn test", pwtest.Score)
	}

	hash := Hash(f.Config.Salt + writePw)
	if hash == f.Config.ReadPassword {
		return fmt.Errorf("read and write passwords cannot match")
	}
	f.Config.WritePassword = hash
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
	data, _ := json.MarshalIndent(&f.Config, "", "  ")
	return ioutil.WriteFile(path.Join(f.Root, f.RelPath, ".razbox"), data, 0755)
}

// EnsureReadAccess returns an error if the access token doesn't permit read access
func (f *Folder) EnsureReadAccess(token *AccessToken) error {
	if len(f.Config.ReadPassword) == 0 {
		return nil
	}

	pw, _ := token.Read[FilenameToUUID(f.RelPath)]
	if pw != f.Config.ReadPassword {
		return fmt.Errorf("incorrect read password")
	}

	return nil
}

// EnsureWriteAccess returns an error if the access token doesn't permit write access
func (f *Folder) EnsureWriteAccess(token *AccessToken) error {
	if len(f.Config.WritePassword) == 0 {
		return fmt.Errorf("folder not writable")
	}

	pw, _ := token.Write[FilenameToUUID(f.RelPath)]
	if pw != f.Config.WritePassword {
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
	if len(f.Config.ReadPassword) == 0 && len(readPw) == 0 {
		return true
	}
	return Hash(f.Config.Salt+readPw) == f.Config.ReadPassword
}

// TestWritePassword returns true if the given password matches the write password
func (f *Folder) TestWritePassword(writePw string) bool {
	return Hash(f.Config.Salt+writePw) == f.Config.WritePassword
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
		return f.Config.ReadPassword, nil
	case "write":
		return f.Config.WritePassword, nil
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
