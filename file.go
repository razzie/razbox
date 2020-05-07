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
	Name     string    `json:"name"`
	RelPath  string    `json:"rel_path,omitempty"`
	Tags     []string  `json:"tags"`
	MIME     string    `json:"mime"`
	Size     int64     `json:"size"`
	Uploaded time.Time `json:"uploaded"`
}

// FileReader ...
type FileReader interface {
	io.Reader
	io.Closer
}

func getFile(relPath string) (*File, error) {
	file := &File{
		RelPath: relPath,
	}

	data, err := ioutil.ReadFile(path.Join(Root, relPath+".json"))
	if err != nil {
		return nil, err
	}

	return file, json.Unmarshal(data, file)
}

// Open ...
func (f *File) Open() (FileReader, error) {
	return os.Open(path.Join(Root, f.RelPath+".bin"))
}

// Save ...
func (f *File) Save() error {
	data, _ := json.MarshalIndent(f, "", "  ")
	return ioutil.WriteFile(path.Join(Root, f.RelPath+".json"), data, 0755)
}

// Create ...
func (f *File) Create(content io.Reader) error {
	jsonFilename := path.Join(Root, f.RelPath+".json")
	if _, err := os.Stat(jsonFilename); os.IsNotExist(err) {
		data, _ := json.MarshalIndent(f, "", "  ")
		err := ioutil.WriteFile(jsonFilename, data, 0755)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("file already exists: %s", f.Name)
	}

	if content != nil {
		file, err := os.Create(path.Join(Root, f.RelPath+".bin"))
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
func (f *File) Move(relPath string) error {
	reader, err := f.Open()
	if err != nil {
		return err
	}
	//defer reader.Close()

	oldName := f.Name
	oldRelPath := f.RelPath
	f.Name = filepath.Base(relPath)
	f.RelPath = path.Join(path.Dir(relPath), FilenameToUUID(f.Name))

	err = f.Create(reader)
	reader.Close()
	if err != nil {
		f.Name = oldName
		f.RelPath = oldRelPath
		return err
	}

	_ = os.Remove(path.Join(Root, oldRelPath+".json"))
	return os.Remove(path.Join(Root, oldRelPath+".bin"))
}

// Delete ...
func (f *File) Delete() error {
	_ = os.Remove(path.Join(Root, f.RelPath+".json"))
	return os.Remove(path.Join(Root, f.RelPath+".bin"))
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
