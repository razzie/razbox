package razbox

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
)

// File ...
type File struct {
	Name     string    `json:"name"`
	RelPath  string    `json:"rel_path,omitempty"`
	Tags     []string  `json:"tags"`
	MIME     string    `json:"mime"`
	Size     int64     `json:"size"`
	Uploaded time.Time `json:"uploaded"`
	Public   bool      `json:"public"`
}

// FileReader ...
type FileReader interface {
	io.Reader
	io.Seeker
	io.Closer
	Stat() (os.FileInfo, error)
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
func (f *File) Create(content io.Reader, overwrite bool) error {
	jsonFilename := path.Join(Root, f.RelPath+".json")
	if _, err := os.Stat(jsonFilename); os.IsNotExist(err) || overwrite {
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
		if err != nil {
			os.Remove(jsonFilename)
			return err
		}
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

	err = f.Create(reader, false)
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

// FixMimeAndSize ...
func (f *File) FixMimeAndSize() error {
	file, err := f.Open()
	if err != nil {
		return err
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return err
	}

	mime, _ := mimetype.DetectReader(file)
	f.Size = fi.Size()
	f.MIME = mime.String()

	return f.Save()
}

// credit: https://github.com/rb-de0/go-mp4-stream/
func (f *File) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const BUFSIZE = 1024 * 8

	file, err := os.Open(path.Join(Root, f.RelPath+".bin"))
	if err != nil {
		w.WriteHeader(500)
		return
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		w.WriteHeader(500)
		return
	}

	fileSize := int(fi.Size())

	if len(r.Header.Get("Range")) == 0 {
		contentLength := strconv.Itoa(fileSize)
		contentEnd := strconv.Itoa(fileSize - 1)

		w.Header().Set("Content-Type", f.MIME)
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Length", contentLength)
		w.Header().Set("Content-Range", "bytes 0-"+contentEnd+"/"+contentLength)
		w.WriteHeader(200)

		buffer := make([]byte, BUFSIZE)

		for {
			n, err := file.Read(buffer)
			if n == 0 {
				break
			}
			if err != nil {
				break
			}

			data := buffer[:n]
			w.Write(data)
			w.(http.Flusher).Flush()
		}
	} else {
		rangeParam := strings.Split(r.Header.Get("Range"), "=")[1]
		splitParams := strings.Split(rangeParam, "-")

		// response values
		contentStartValue := 0
		contentStart := strconv.Itoa(contentStartValue)
		contentEndValue := fileSize - 1
		contentEnd := strconv.Itoa(contentEndValue)
		contentSize := strconv.Itoa(fileSize)

		if len(splitParams) > 0 {
			contentStartValue, err = strconv.Atoi(splitParams[0])
			if err != nil {
				contentStartValue = 0
			}
			contentStart = strconv.Itoa(contentStartValue)
		}

		if len(splitParams) > 1 {
			contentEndValue, err = strconv.Atoi(splitParams[1])
			if err != nil {
				contentEndValue = fileSize - 1
			}
			contentEnd = strconv.Itoa(contentEndValue)
		}

		contentLength := strconv.Itoa(contentEndValue - contentStartValue + 1)

		w.Header().Set("Content-Type", f.MIME)
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Length", contentLength)
		w.Header().Set("Content-Range", "bytes "+contentStart+"-"+contentEnd+"/"+contentSize)
		w.WriteHeader(206)

		buffer := make([]byte, BUFSIZE)

		file.Seek(int64(contentStartValue), 0)

		writeBytes := 0

		for {
			n, err := file.Read(buffer)
			writeBytes += n
			if n == 0 {
				break
			}
			if err != nil {
				break
			}

			if writeBytes >= contentEndValue {
				data := buffer[:BUFSIZE-writeBytes+contentEndValue+1]
				w.Write(data)
				w.(http.Flusher).Flush()
				break
			}

			data := buffer[:n]
			w.Write(data)
			w.(http.Flusher).Flush()
		}
	}
}
