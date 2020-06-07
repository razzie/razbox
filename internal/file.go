package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/gabriel-vasile/mimetype"
)

// FileReader ...
type FileReader interface {
	io.Reader
	io.Seeker
	io.Closer
	Stat() (os.FileInfo, error)
}

// File ...
type File struct {
	Name      string     `json:"name"`
	Root      string     `json:"root"`
	RelPath   string     `json:"rel_path"`
	Tags      []string   `json:"tags"`
	MIME      string     `json:"mime"`
	Size      int64      `json:"size"`
	Uploaded  time.Time  `json:"uploaded"`
	Public    bool       `json:"public"`
	Thumbnail *Thumbnail `json:"thumbnail,omitempty"`
}

func getFile(root, relPath string) (*File, error) {
	file := &File{
		Root:    root,
		RelPath: relPath,
	}

	data, err := ioutil.ReadFile(path.Join(root, relPath+".json"))
	if err != nil {
		return nil, err
	}

	return file, json.Unmarshal(data, file)
}

// Open ...
func (f *File) Open() (FileReader, error) {
	return os.Open(path.Join(f.Root, f.RelPath+".bin"))
}

// Save ...
func (f *File) Save() error {
	data, _ := json.MarshalIndent(f, "", "  ")
	return ioutil.WriteFile(path.Join(f.Root, f.RelPath+".json"), data, 0755)
}

// Create ...
func (f *File) Create(content io.Reader, overwrite bool) error {
	absPath := path.Join(f.Root, f.RelPath)
	dataFilename := absPath + ".bin"
	jsonFilename := absPath + ".json"

	if _, err := os.Stat(jsonFilename); os.IsNotExist(err) || overwrite {
		data, _ := json.MarshalIndent(f, "", "  ")
		err := ioutil.WriteFile(jsonFilename, data, 0755)
		if err != nil {
			return err
		}
	} else {
		return &ErrFileAlreadyExists{File: f.Name}
	}

	if content != nil {
		tmpfile, err := ioutil.TempFile(f.Root, fmt.Sprintf("razbox-upload-*-%s", f.Name))
		if err != nil {
			return err
		}
		defer os.Remove(tmpfile.Name())
		defer tmpfile.Close()

		n, err := io.Copy(tmpfile, content)
		if err != nil {
			os.Remove(jsonFilename)
			return err
		}

		if len(f.MIME) == 0 || f.Size == 0 {
			tmpfile.Seek(0, io.SeekStart)
			mime, _ := mimetype.DetectReader(tmpfile)
			f.MIME = mime.String()
			f.Size = n
			f.Save()
		}

		tmpfile.Close()
		err = os.Rename(tmpfile.Name(), dataFilename)
		if err != nil {
			os.Remove(jsonFilename)
			return err
		}
	}

	return nil
}

// Move ...
func (f *File) Move(relPath string) error {
	oldName := f.Name
	oldRelPath := f.RelPath
	f.Name = filepath.Base(relPath)
	f.RelPath = path.Join(path.Dir(relPath), FilenameToUUID(f.Name))

	err := f.Create(nil, false)
	if err != nil {
		f.Name = oldName
		f.RelPath = oldRelPath
		return err
	}

	err = os.Rename(
		path.Join(f.Root, oldRelPath+".bin"),
		path.Join(f.Root, f.RelPath+".bin"),
	)
	if err != nil {
		_ = os.Remove(path.Join(f.Root, f.RelPath+".json"))
		f.Name = oldName
		f.RelPath = oldRelPath
		return err
	}

	_ = os.Remove(path.Join(f.Root, oldRelPath+".json"))
	return nil
}

// Delete ...
func (f *File) Delete() error {
	_ = os.Remove(path.Join(f.Root, f.RelPath+".json"))
	return os.Remove(path.Join(f.Root, f.RelPath+".bin"))
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
