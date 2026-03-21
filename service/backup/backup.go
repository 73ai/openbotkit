package backup

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"

	"github.com/73ai/openbotkit/config"
)

type RunResult struct {
	Changed  int
	Skipped  int
	Uploaded int64
	Duration time.Duration
}

type Service struct {
	backend      Backend
	baseDir      string
	manifestPath string
	stagingDir   string
}

func New(backend Backend, baseDir string) *Service {
	return &Service{
		backend:      backend,
		baseDir:      baseDir,
		manifestPath: config.BackupLastManifestPath(),
		stagingDir:   config.BackupStagingDir(),
	}
}

func NewWithPaths(backend Backend, baseDir, manifestPath, stagingDir string) *Service {
	return &Service{
		backend:      backend,
		baseDir:      baseDir,
		manifestPath: manifestPath,
		stagingDir:   stagingDir,
	}
}

func (s *Service) Run(ctx context.Context) (*RunResult, error) {
	start := time.Now()

	lastManifest, err := LoadManifest(s.manifestPath)
	if err != nil {
		return nil, fmt.Errorf("load last manifest: %w", err)
	}

	files, err := ScanFiles(s.baseDir)
	if err != nil {
		return nil, fmt.Errorf("scan files: %w", err)
	}

	stagingDir := s.stagingDir
	if err := os.MkdirAll(stagingDir, 0700); err != nil {
		return nil, fmt.Errorf("create staging dir: %w", err)
	}
	defer os.RemoveAll(stagingDir)

	hashes := make(map[string]string)
	stagedPaths := make(map[string]string)

	for _, rel := range files {
		absPath := filepath.Join(s.baseDir, rel)
		var filePath string

		if strings.HasSuffix(rel, ".db") {
			vacuumed, err := VacuumInto(absPath, stagingDir, rel)
			if err != nil {
				return nil, fmt.Errorf("vacuum %s: %w", rel, err)
			}
			filePath = vacuumed
		} else {
			filePath = absPath
		}

		hash, err := hashFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("hash %s: %w", rel, err)
		}
		hashes[rel] = "sha256:" + hash
		stagedPaths[rel] = filePath
	}

	diff := DiffManifest(lastManifest, hashes)

	hostname, _ := os.Hostname()
	manifest := NewManifest(hostname)

	var result RunResult

	for _, rel := range diff.Changed {
		filePath := stagedPaths[rel]
		hash := hashes[rel]

		compressed, err := compressFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("compress %s: %w", rel, err)
		}

		objectKey := objectKeyFromHash(hash)
		exists, err := s.backend.Head(ctx, objectKey)
		if err != nil {
			return nil, fmt.Errorf("check object %s: %w", objectKey, err)
		}

		if !exists {
			reader := bytes.NewReader(compressed)
			if err := s.backend.Put(ctx, objectKey, reader, int64(len(compressed))); err != nil {
				return nil, fmt.Errorf("upload %s: %w", rel, err)
			}
			result.Uploaded += int64(len(compressed))
		}

		info, err := os.Stat(filePath)
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", rel, err)
		}

		manifest.Files[rel] = ManifestFile{
			Hash:           hash,
			Size:           info.Size(),
			CompressedSize: int64(len(compressed)),
		}
		result.Changed++
	}

	for rel, hash := range hashes {
		if _, ok := manifest.Files[rel]; ok {
			continue
		}
		prev, ok := lastManifest.Files[rel]
		if ok {
			manifest.Files[rel] = prev
		} else {
			info, _ := os.Stat(stagedPaths[rel])
			size := int64(0)
			if info != nil {
				size = info.Size()
			}
			manifest.Files[rel] = ManifestFile{
				Hash: hash,
				Size: size,
			}
		}
		result.Skipped++
	}

	manifestKey := fmt.Sprintf("snapshots/%s.json", manifest.ID)
	manifestData, err := marshalManifest(manifest)
	if err != nil {
		return nil, fmt.Errorf("marshal manifest: %w", err)
	}
	if err := s.backend.Put(ctx, manifestKey, bytes.NewReader(manifestData), int64(len(manifestData))); err != nil {
		return nil, fmt.Errorf("upload manifest: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(s.manifestPath), 0700); err != nil {
		return nil, fmt.Errorf("create backup dir: %w", err)
	}
	if err := SaveManifest(s.manifestPath, manifest); err != nil {
		return nil, fmt.Errorf("save local manifest: %w", err)
	}

	result.Duration = time.Since(start)
	return &result, nil
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func compressFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	w, err := zstd.NewWriter(&buf)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(data); err != nil {
		w.Close()
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func objectKeyFromHash(hash string) string {
	hex := strings.TrimPrefix(hash, "sha256:")
	return fmt.Sprintf("objects/%s/%s", hex[:2], hex)
}

func marshalManifest(m *Manifest) ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}
