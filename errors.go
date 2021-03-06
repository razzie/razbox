package razbox

import (
	"fmt"
	"net/http"
)

// ErrNotFound ...
type ErrNotFound struct{}

func (err ErrNotFound) Error() string {
	return "Not found"
}

// ErrNoReadAccess ...
type ErrNoReadAccess struct {
	Folder string
}

func (err ErrNoReadAccess) Error() string {
	return err.Folder + ": no read access"
}

// ErrNoWriteAccess ...
type ErrNoWriteAccess struct {
	Folder string
}

func (err ErrNoWriteAccess) Error() string {
	return err.Folder + ": no write access"
}

// ErrWrongPassword ...
type ErrWrongPassword struct{}

func (err ErrWrongPassword) Error() string {
	return "Wrong password"
}

// ErrSizeLimitExceeded ...
type ErrSizeLimitExceeded struct{}

func (err ErrSizeLimitExceeded) Error() string {
	return "Size limit exceeded"
}

// ErrUnsupportedFileFormat ...
type ErrUnsupportedFileFormat struct {
	MIME string
}

func (err ErrUnsupportedFileFormat) Error() string {
	return "Unsupported file format: " + err.MIME
}

// ErrBadHTTPResponseStatus ...
type ErrBadHTTPResponseStatus struct {
	StatusCode int
}

func (err ErrBadHTTPResponseStatus) Error() string {
	return "bad response status code: " + http.StatusText(err.StatusCode)
}

// ErrInvalidName ...
type ErrInvalidName struct {
	Name string
}

func (err ErrInvalidName) Error() string {
	return "Invalid name: " + err.Name
}

// ErrInvalidMoveLocation ...
type ErrInvalidMoveLocation struct {
	Location string
}

func (err ErrInvalidMoveLocation) Error() string {
	return "Invalid move location: " + err.Location
}

// ErrNotDeletable ...
type ErrNotDeletable struct {
	Name string
}

func (err ErrNotDeletable) Error() string {
	return "Not deletable: " + err.Name
}

// ErrRateLimitExceeded ...
type ErrRateLimitExceeded struct {
	ReqPerMin int
}

func (err ErrRateLimitExceeded) Error() string {
	msg := "Rate limit exceeded"
	if err.ReqPerMin > 0 {
		msg += fmt.Sprintf(" (%d / minute)", err.ReqPerMin)
	}
	return msg
}

// ErrNoFiles ...
type ErrNoFiles struct{}

func (err ErrNoFiles) Error() string {
	return "No files"
}

// ErrSubfoldersDisabled ...
type ErrSubfoldersDisabled struct {
	Folder string
}

func (err ErrSubfoldersDisabled) Error() string {
	return "Subfolders are disabled for this folder: " + err.Folder
}

// ErrFolderBusy ...
type ErrFolderBusy struct{}

func (err ErrFolderBusy) Error() string {
	return "Folder is busy"
}
