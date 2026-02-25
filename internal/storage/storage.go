package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type Service struct {
	rootDir string
}

func NewService(rootDir string) (*Service, error) {
	abs, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("resolve upload dir %s: %w", rootDir, err)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, fmt.Errorf("create upload dir %s: %w", abs, err)
	}
	return &Service{rootDir: abs}, nil
}

// Save writes the contents of r to disk and returns the storage key (relative path).
// For backups (category="backup"), files go under _system/backups/.
// For everything else, files go under {companyID}/{category}/{entityID}/.
func (s *Service) Save(companyID int, category string, entityID int, ext string, r io.Reader) (string, int64, error) {
	id := uuid.New().String()
	filename := id + ext

	var rel string
	if category == "backup" {
		rel = filepath.Join("_system", "backups", filename)
	} else {
		rel = filepath.Join(fmt.Sprintf("%d", companyID), category, fmt.Sprintf("%d", entityID), filename)
	}

	abs, err := s.safePath(rel)
	if err != nil {
		return "", 0, err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", 0, fmt.Errorf("create dir: %w", err)
	}

	f, err := os.Create(abs)
	if err != nil {
		return "", 0, fmt.Errorf("create file: %w", err)
	}

	written, err := io.Copy(f, r)
	if err != nil {
		f.Close()
		os.Remove(abs)
		return "", 0, fmt.Errorf("write file: %w", err)
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(abs)
		return "", 0, fmt.Errorf("sync file: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(abs)
		return "", 0, fmt.Errorf("close file: %w", err)
	}

	return rel, written, nil
}

// Open returns a reader for the file at the given storage key.
func (s *Service) Open(storageKey string) (io.ReadCloser, error) {
	abs, err := s.safePath(storageKey)
	if err != nil {
		return nil, err
	}
	return os.Open(abs)
}

// Delete removes the file at the given storage key from disk.
func (s *Service) Delete(storageKey string) error {
	abs, err := s.safePath(storageKey)
	if err != nil {
		return err
	}
	return os.Remove(abs)
}

// safePath resolves storageKey relative to rootDir and validates
// that the result stays within rootDir (prevents path traversal).
func (s *Service) safePath(storageKey string) (string, error) {
	if storageKey == "" {
		return "", fmt.Errorf("empty storage key")
	}
	abs := filepath.Clean(filepath.Join(s.rootDir, storageKey))
	if !strings.HasPrefix(abs, s.rootDir+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid storage key: path traversal")
	}
	return abs, nil
}
