package internal

import (
	"net/http"
	"os"
	"time"
)

// FileReader ...
type FileReader interface {
	http.File
	os.FileInfo
	MimeType() string
}

type fileReader struct {
	file *os.File
	sys  *File
}

func (r fileReader) Read(p []byte) (int, error) {
	return r.file.Read(p)
}

func (r fileReader) Seek(offset int64, whence int) (int64, error) {
	return r.file.Seek(offset, whence)
}

func (r fileReader) Close() error {
	return r.file.Close()
}

func (r *fileReader) Stat() (os.FileInfo, error) {
	return r, nil
}

func (r fileReader) Readdir(count int) ([]os.FileInfo, error) {
	return nil, nil
}

func (r fileReader) Name() string {
	return r.sys.Name
}

func (r fileReader) Size() int64 {
	return r.sys.Size
}

func (r fileReader) Mode() os.FileMode {
	return os.ModePerm
}

func (r fileReader) ModTime() time.Time {
	return r.sys.Uploaded
}

func (r fileReader) IsDir() bool {
	return false
}

func (r fileReader) Sys() interface{} {
	return r.sys
}

func (r fileReader) MimeType() string {
	return r.sys.MIME
}

func newFileReader(sys *File) (*fileReader, error) {
	file, err := os.Open(sys.GetInternalFilename())
	if err != nil {
		return nil, err
	}
	return &fileReader{
		file: file,
		sys:  sys,
	}, nil
}
