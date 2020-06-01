package internal

import (
	"fmt"
)

// ErrFolderMissingCache ...
type ErrFolderMissingCache struct {
	Folder string
}

func (err ErrFolderMissingCache) Error() string {
	return fmt.Sprintf("Cached folder %s doesn't contain cached file or subfolder list", err.Folder)
}

// ErrFileAlreadyExists ...
type ErrFileAlreadyExists struct {
	File string
}

func (err ErrFileAlreadyExists) Error() string {
	return "File already exists: " + err.File
}

// ErrFolderConfigNotFound ...
type ErrFolderConfigNotFound struct {
	Folder string
}

func (err ErrFolderConfigNotFound) Error() string {
	return "Config file not found for folder: " + err.Folder
}

// ErrInheritedConfigPasswordChange ...
type ErrInheritedConfigPasswordChange struct{}

func (err ErrInheritedConfigPasswordChange) Error() string {
	return "Cannot change password of folders that inherit parent configuration"
}

// ErrReadWritePasswordMatch ...
type ErrReadWritePasswordMatch struct{}

func (err ErrReadWritePasswordMatch) Error() string {
	return "Read and write passwords cannot match"
}

// ErrWrongPassword ...
type ErrWrongPassword struct{}

func (err ErrWrongPassword) Error() string {
	return "Wrong password"
}

// ErrPasswordScoreTooLow ...
type ErrPasswordScoreTooLow struct {
	Score int
}

func (err ErrPasswordScoreTooLow) Error() string {
	return fmt.Sprintf("Password scored too low (%d) on zxcvbn test", err.Score)
}

// ErrInvalidAccessType ...
type ErrInvalidAccessType struct {
	AccessType string
}

func (err ErrInvalidAccessType) Error() string {
	return "Invalid access type: " + err.AccessType
}

// ErrFolderNotWritable ...
type ErrFolderNotWritable struct{}

func (err ErrFolderNotWritable) Error() string {
	return "Folder not writable"
}

// ErrUnsupportedFileFormat ...
type ErrUnsupportedFileFormat struct {
	MIME string
}

func (err ErrUnsupportedFileFormat) Error() string {
	return "Unsupported file format: " + err.MIME
}
