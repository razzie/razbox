package razlink

// Entry ...
type Entry struct {
	URL          string
	Method       ServeMethod
	Salt         string
	PasswordHash string
	Permanent    bool
}

// NewEntry ...
func NewEntry(url, password string, method ServeMethod) *Entry {
	salt := Hash(url)

	return &Entry{
		URL:          url,
		Method:       method,
		Salt:         salt,
		PasswordHash: Hash(salt + password),
	}
}

// MatchPassword ...
func (entry *Entry) MatchPassword(password string) bool {
	return entry.PasswordHash == Hash(entry.Salt+password)
}

// SetPassword ...
func (entry *Entry) SetPassword(password string) {
	entry.PasswordHash = Hash(entry.Salt + password)
}
