package razbox

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
