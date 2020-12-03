package internal

import (
	"encoding/json"
	"fmt"
	"image"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"
)

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
	data, err := ioutil.ReadFile(path.Join(root, relPath+".json"))
	if err != nil {
		return nil, err
	}

	f := &File{
		Root:    root,
		RelPath: relPath,
	}

	if err := json.Unmarshal(data, f); err != nil {
		return nil, err
	}
	f.Root = root // overwrite root with possible new root
	return f, nil
}

// GetInternalFilename ...
func (f *File) GetInternalFilename() string {
	return path.Join(f.Root, f.RelPath+".bin")
}

// GetThumbnail ...
func (f *File) GetThumbnail(retryAfter time.Duration) (*Thumbnail, error) {
	if !IsThumbnailSupported(f.MIME) {
		return nil, &ErrUnsupportedFileFormat{MIME: f.MIME}
	}

	thumbFilename := path.Join(f.Root, f.RelPath+".thumb")

	// migrate thumbnails to .thumb files
	if f.Thumbnail != nil && len(f.Thumbnail.Data) > 0 {
		data, _ := json.MarshalIndent(f.Thumbnail, "", "  ")
		if err := ioutil.WriteFile(thumbFilename, data, 0644); err != nil {
			return f.Thumbnail, err
		}
		thumb := f.Thumbnail
		f.Thumbnail.Data = nil
		f.Save()
		return thumb, nil
	}

	data, err := ioutil.ReadFile(thumbFilename)
	if err != nil {
		if os.IsNotExist(err) {
			return f.createThumbnail()
		}
		return nil, err
	}
	thumb := new(Thumbnail)
	if err := json.Unmarshal(data, &thumb); err != nil {
		return nil, err
	}
	if len(thumb.Data) == 0 && thumb.Timestamp.Add(retryAfter).Before(time.Now()) {
		return f.createThumbnail()
	}
	return thumb, nil
}

// GetThumbnailBounds ...
func (f *File) GetThumbnailBounds(retryAfter time.Duration) (*image.Rectangle, error) {
	if f.Thumbnail != nil {
		return &f.Thumbnail.Bounds, nil
	}
	thumb, err := f.GetThumbnail(retryAfter)
	if err != nil {
		return nil, err
	}
	return &thumb.Bounds, nil
}

func (f *File) createThumbnail() (*Thumbnail, error) {
	thumbFilename := path.Join(f.Root, f.RelPath+".thumb")
	thumb, err := GetThumbnail(f.GetInternalFilename(), f.MIME)
	if err != nil {
		thumb = &Thumbnail{Timestamp: time.Now()}
		data, _ := json.MarshalIndent(thumb, "", "  ")
		ioutil.WriteFile(thumbFilename, data, 0644)
		return nil, err
	}
	data, _ := json.MarshalIndent(thumb, "", "  ")
	ioutil.WriteFile(thumbFilename, data, 0644)
	f.Thumbnail = &Thumbnail{
		MIME:      thumb.MIME,
		Bounds:    thumb.Bounds,
		Timestamp: thumb.Timestamp,
	}
	f.Save()
	return thumb, nil
}

// Open ...
func (f *File) Open() (FileReader, error) {
	return newFileReader(f)
}

// Save ...
func (f *File) Save() error {
	data, _ := json.MarshalIndent(f, "", "  ")
	return ioutil.WriteFile(path.Join(f.Root, f.RelPath+".json"), data, 0644)
}

// Create ...
func (f *File) Create(content io.Reader, overwrite bool) error {
	absPath := path.Join(f.Root, f.RelPath)
	dataFilename := absPath + ".bin"
	jsonFilename := absPath + ".json"

	if _, err := os.Stat(jsonFilename); os.IsNotExist(err) || overwrite {
		data, _ := json.MarshalIndent(f, "", "  ")
		err := ioutil.WriteFile(jsonFilename, data, 0644)
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
			f.MIME, _ = DetectContentType(tmpfile)
			f.Size = n
			f.Save()
		}

		tmpfile.Close()
		os.Chmod(tmpfile.Name(), 0644)
		err = os.Rename(tmpfile.Name(), dataFilename)
		if err != nil {
			os.Remove(jsonFilename)
			return err
		}

		if IsThumbnailSupported(f.MIME) {
			f.createThumbnail()
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
