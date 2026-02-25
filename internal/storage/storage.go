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
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return nil, fmt.Errorf("create upload dir %s: %w", rootDir, err)
	}
	return &Service{rootDir: rootDir}, nil
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

	abs := filepath.Join(s.rootDir, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", 0, fmt.Errorf("create dir: %w", err)
	}

	f, err := os.Create(abs)
	if err != nil {
		return "", 0, fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	written, err := io.Copy(f, r)
	if err != nil {
		os.Remove(abs)
		return "", 0, fmt.Errorf("write file: %w", err)
	}

	return rel, written, nil
}

// Open returns a reader for the file at the given storage key.
// It validates the key to prevent path traversal.
func (s *Service) Open(storageKey string) (io.ReadCloser, error) {
	if err := validateKey(storageKey); err != nil {
		return nil, err
	}
	abs := filepath.Join(s.rootDir, storageKey)
	return os.Open(abs)
}

// Delete removes the file at the given storage key from disk.
func (s *Service) Delete(storageKey string) error {
	if err := validateKey(storageKey); err != nil {
		return err
	}
	abs := filepath.Join(s.rootDir, storageKey)
	return os.Remove(abs)
}

func validateKey(key string) error {
	if key == "" {
		return fmt.Errorf("empty storage key")
	}
	if strings.Contains(key, "..") {
		return fmt.Errorf("invalid storage key: path traversal")
	}
	if filepath.IsAbs(key) {
		return fmt.Errorf("invalid storage key: absolute path")
	}
	return nil
}
