package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

type Client interface {
	Save(ctx context.Context, personID uuid.UUID, filename string, reader io.Reader) (string, error)
	Open(ctx context.Context, storagePath string) (*os.File, error)
	Delete(ctx context.Context, storagePath string) error
}

type client struct {
	rootDir string
}

func New(rootDir string) (Client, error) {
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return nil, err
	}
	return &client{rootDir: rootDir}, nil
}

func (c *client) Save(ctx context.Context, personID uuid.UUID, filename string, reader io.Reader) (string, error) {
	_ = ctx

	dir := filepath.Join(c.rootDir, personID.String())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	storagePath := filepath.Join(personID.String(), uuid.NewString()+"_"+filepath.Base(filename))
	fullPath := filepath.Join(c.rootDir, storagePath)

	file, err := os.Create(fullPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := io.Copy(file, reader); err != nil {
		_ = os.Remove(fullPath)
		return "", err
	}

	return storagePath, nil
}

func (c *client) Open(ctx context.Context, storagePath string) (*os.File, error) {
	_ = ctx
	fullPath := filepath.Join(c.rootDir, filepath.Clean(storagePath))
	return os.Open(fullPath)
}

func (c *client) Delete(ctx context.Context, storagePath string) error {
	_ = ctx
	fullPath := filepath.Join(c.rootDir, filepath.Clean(storagePath))
	return os.Remove(fullPath)
}