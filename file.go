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
