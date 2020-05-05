package razbox

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
)

// File ...
type File struct {
	Name         string   `json:"name"`
	InternalName string   `json:"-"`
	Tags         []string `json:"tags"`
	MIME         string   `json:"mime"`
	Size         string   `json:"size"`
	Uploaded     string   `json:"uploaded"`
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
	data, _ := json.MarshalIndent(f, "", "  ")
	err := ioutil.WriteFile(f.InternalName+".json", data, 0644)
	if err != nil {
		return err
	}

	file, err := os.Create(f.InternalName + ".bin")
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, content)
	return err
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
