package backup

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Backend interface {
	Put(ctx context.Context, key string, r io.Reader, size int64) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Head(ctx context.Context, key string) (bool, error)
	List(ctx context.Context, prefix string) ([]string, error)
	Delete(ctx context.Context, key string) error
}

type LocalBackend struct {
	root string
}

func NewLocalBackend(root string) *LocalBackend {
	return &LocalBackend{root: root}
}

func (b *LocalBackend) Put(_ context.Context, key string, r io.Reader, _ int64) error {
	path := filepath.Join(b.root, key)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create dir for %s: %w", key, err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", key, err)
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("write %s: %w", key, err)
	}
	return nil
}

func (b *LocalBackend) Get(_ context.Context, key string) (io.ReadCloser, error) {
	path := filepath.Join(b.root, key)
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", key, err)
	}
	return f, nil
}

func (b *LocalBackend) Head(_ context.Context, key string) (bool, error) {
	path := filepath.Join(b.root, key)
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (b *LocalBackend) List(_ context.Context, prefix string) ([]string, error) {
	dir := filepath.Join(b.root, prefix)
	var keys []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(b.root, path)
		if err != nil {
			return err
		}
		keys = append(keys, strings.ReplaceAll(rel, string(filepath.Separator), "/"))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", prefix, err)
	}
	sort.Strings(keys)
	return keys, nil
}

func (b *LocalBackend) Delete(_ context.Context, key string) error {
	path := filepath.Join(b.root, key)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete %s: %w", key, err)
	}
	return nil
}

var _ Backend = (*LocalBackend)(nil)
