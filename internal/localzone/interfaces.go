package localzone

import (
	"errors"
	"io/fs"
	"os"
)

// FileReader interface for reading files (for testing).
type FileReader interface {
	ReadFile(path string) ([]byte, error)
	FileExists(path string) bool
}

// ZoneDetectorInterface interface for detecting zones.
type ZoneDetectorInterface interface {
	DetectZones() ([]string, error)
}

// FileWatcher interface for watching file changes.
type FileWatcher interface {
	Watch(paths []string, callback func()) error
	Close() error
}

// OSFileReader implements FileReader using real OS files.
type OSFileReader struct{}

func (r *OSFileReader) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path) //nolint:gosec
}

func (r *OSFileReader) FileExists(path string) bool {
	_, err := os.Stat(path)

	return !errors.Is(err, fs.ErrNotExist)
}

// MockFileReader implements FileReader for testing.
type MockFileReader struct {
	Files map[string][]byte
}

func NewMockFileReader() *MockFileReader {
	return &MockFileReader{
		Files: make(map[string][]byte),
	}
}

func (r *MockFileReader) ReadFile(path string) ([]byte, error) {
	if content, exists := r.Files[path]; exists {
		return content, nil
	}

	return nil, os.ErrNotExist
}

func (r *MockFileReader) FileExists(path string) bool {
	_, exists := r.Files[path]

	return exists
}

func (r *MockFileReader) SetFile(path string, content []byte) {
	r.Files[path] = content
}
