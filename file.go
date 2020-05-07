package razbox

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"
)

// File ...
type File struct {
	Name         string    `json:"name"`
	InternalName string    `json:"internal_name,omitempty"`
	Tags         []string  `json:"tags"`
	MIME         string    `json:"mime"`
	Size         string    `json:"size"`
	Uploaded     time.Time `json:"uploaded"`
}

// FileReader ...
type FileReader interface {
	io.Reader
	io.Closer
}

// GetFile ...
func GetFile(path string) (*File, error) {
	file := &File{
		InternalName: path,
	}
	data, err := ioutil.ReadFile(path + ".json")
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
func (f *File) Save(content io.Reader) error {
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
	reader, err := f.Open()
	if err != nil {
		return err
	}
	//defer reader.Close()

	oldName := f.Name
	oldInternalName := f.InternalName
	f.Name = filepath.Base(newNameAndPath)
	f.InternalName = path.Join(path.Dir(newNameAndPath), FilenameToUUID(f.Name))

	err = f.Save(reader)
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
