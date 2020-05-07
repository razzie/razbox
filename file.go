package razbox

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// File ...
type File struct {
	Name         string    `json:"name"`
	InternalName string    `json:"internal_name,omitempty"`
	Tags         []string  `json:"tags"`
	MIME         string    `json:"mime"`
	Size         int64     `json:"size"`
	Uploaded     time.Time `json:"uploaded"`
}

// FileReader ...
type FileReader interface {
	io.Reader
	io.Closer
}

// GetFile ...
func GetFile(uri string) (*File, error) {
	if !filepath.IsAbs(uri) {
		uri = path.Join(Root, uri)
	}

	if !strings.HasPrefix(uri, Root) {
		return nil, fmt.Errorf("path %s is not in root (%s)", uri, Root)
	}

	file := &File{
		InternalName: uri,
	}

	data, err := ioutil.ReadFile(uri + ".json")
	if err != nil {
		return nil, err
	}

	return file, json.Unmarshal(data, file)
}

// Open ...
func (f *File) Open() (FileReader, error) {
	return os.Open(f.InternalName + ".bin")
}

// Save ...
func (f *File) Save() error {
	data, _ := json.MarshalIndent(f, "", "  ")
	return ioutil.WriteFile(f.InternalName+".json", data, 0644)
}

// Create ...
func (f *File) Create(content io.Reader) error {
	jsonFilename := f.InternalName + ".json"
	if _, err := os.Stat(jsonFilename); os.IsNotExist(err) {
		data, _ := json.MarshalIndent(f, "", "  ")
		err := ioutil.WriteFile(jsonFilename, data, 0644)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("file already exists: %s", jsonFilename)
	}

	if content != nil {
		file, err := os.Create(f.InternalName + ".bin")
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(file, content)
		return err
	}

	return nil
}

// Move ...
func (f *File) Move(newNameAndPath string) error {
	if !filepath.IsAbs(newNameAndPath) {
		newNameAndPath = path.Join(Root, newNameAndPath)
	}

	if !strings.HasPrefix(newNameAndPath, Root) {
		return fmt.Errorf("path %s is not in root (%s)", newNameAndPath, Root)
	}

	reader, err := f.Open()
	if err != nil {
		return err
	}
	//defer reader.Close()

	oldName := f.Name
	oldInternalName := f.InternalName
	f.Name = filepath.Base(newNameAndPath)
	f.InternalName = path.Join(path.Dir(newNameAndPath), FilenameToUUID(f.Name))

	err = f.Create(reader)
	reader.Close()
	if err != nil {
		f.Name = oldName
		f.InternalName = oldInternalName
		return err
	}

	_ = os.Remove(oldInternalName + ".json")
	return os.Remove(oldInternalName + ".bin")
}

// Delete ...
func (f *File) Delete() error {
	_ = os.Remove(f.InternalName + ".json")
	return os.Remove(f.InternalName + ".bin")
}

// HasTag ...
func (f *File) HasTag(tag string) bool {
	for _, t := range f.Tags {
		if t == tag {
			return true
		}
	}
	return false
}
