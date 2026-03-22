package backup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/klauspost/compress/zstd"
)

type RestoreResult struct {
	Restored int
}

func (s *Service) Restore(ctx context.Context, snapshotID string) (*RestoreResult, error) {
	manifestKey := fmt.Sprintf("snapshots/%s.json", snapshotID)
	rc, err := s.backend.Get(ctx, manifestKey)
	if err != nil {
		return nil, fmt.Errorf("download manifest: %w", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}

	var result RestoreResult
	for rel, mf := range manifest.Files {
		objectKey := objectKeyFromHash(mf.Hash)
		objRC, err := s.backend.Get(ctx, objectKey)
		if err != nil {
			return nil, fmt.Errorf("download %s: %w", rel, err)
		}

		compressed, err := io.ReadAll(objRC)
		objRC.Close()
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", rel, err)
		}

		decompressed, err := decompressData(compressed)
		if err != nil {
			return nil, fmt.Errorf("decompress %s: %w", rel, err)
		}

		destPath := filepath.Join(s.baseDir, rel)
		if err := os.MkdirAll(filepath.Dir(destPath), 0700); err != nil {
			return nil, fmt.Errorf("create dir for %s: %w", rel, err)
		}
		if err := os.WriteFile(destPath, decompressed, 0600); err != nil {
			return nil, fmt.Errorf("write %s: %w", rel, err)
		}
		result.Restored++
	}

	return &result, nil
}

func (s *Service) ListSnapshots(ctx context.Context) ([]string, error) {
	keys, err := s.backend.List(ctx, "snapshots/")
	if err != nil {
		return nil, fmt.Errorf("list snapshots: %w", err)
	}

	var ids []string
	for _, key := range keys {
		name := filepath.Base(key)
		ext := filepath.Ext(name)
		if ext == ".json" {
			ids = append(ids, name[:len(name)-len(ext)])
		}
	}
	return ids, nil
}

func (s *Service) GetManifest(ctx context.Context, snapshotID string) (*Manifest, error) {
	manifestKey := fmt.Sprintf("snapshots/%s.json", snapshotID)
	rc, err := s.backend.Get(ctx, manifestKey)
	if err != nil {
		return nil, fmt.Errorf("download manifest: %w", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	return &manifest, nil
}

func decompressData(compressed []byte) ([]byte, error) {
	r, err := zstd.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}
