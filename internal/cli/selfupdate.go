package cli

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const releaseRepo = "73ai/openbotkit"

func releaseAssetName(tag, goos, goarch string) string {
	return fmt.Sprintf("openbotkit_%s_%s_%s.tar.gz", strings.TrimPrefix(tag, "v"), goos, goarch)
}

func checksumsAssetName(tag string) string {
	return fmt.Sprintf("openbotkit_%s_checksums.txt", strings.TrimPrefix(tag, "v"))
}

func verifyChecksum(data []byte, name string, checksums []byte) error {
	scanner := bufio.NewScanner(bytes.NewReader(checksums))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 2 || fields[1] != name {
			continue
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != fields[0] {
			return fmt.Errorf("checksum mismatch for %s", name)
		}
		return nil
	}
	return fmt.Errorf("no checksum listed for %s", name)
}

func extractBinary(archive []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("%s not found in archive", name)
		}
		if err != nil {
			return nil, err
		}
		if hdr.Typeflag == tar.TypeReg && filepath.Base(hdr.Name) == name {
			return io.ReadAll(tr)
		}
	}
}

func replaceBinary(target string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(target), ".obk-update-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o755); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), target)
}

func runningBinaryPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

func httpGet(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func latestReleaseTag() (string, error) {
	body, err := httpGet("https://api.github.com/repos/" + releaseRepo + "/releases/latest")
	if err != nil {
		return "", err
	}
	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(body, &release); err != nil {
		return "", err
	}
	if release.TagName == "" {
		return "", fmt.Errorf("no release tag found")
	}
	return release.TagName, nil
}

func selfUpdate() (string, bool, error) {
	tag, err := latestReleaseTag()
	if err != nil {
		return "", false, fmt.Errorf("find latest release: %w", err)
	}
	if strings.TrimPrefix(tag, "v") == strings.TrimPrefix(Version, "v") {
		return tag, false, nil
	}
	base := "https://github.com/" + releaseRepo + "/releases/download/" + tag + "/"
	asset := releaseAssetName(tag, runtime.GOOS, runtime.GOARCH)
	archive, err := httpGet(base + asset)
	if err != nil {
		return "", false, err
	}
	checksums, err := httpGet(base + checksumsAssetName(tag))
	if err != nil {
		return "", false, err
	}
	if err := verifyChecksum(archive, asset, checksums); err != nil {
		return "", false, err
	}
	bin, err := extractBinary(archive, "obk")
	if err != nil {
		return "", false, err
	}
	target, err := runningBinaryPath()
	if err != nil {
		return "", false, fmt.Errorf("locate running binary: %w", err)
	}
	if err := replaceBinary(target, bin); err != nil {
		return "", false, fmt.Errorf("replace %s: %w", target, err)
	}
	return tag, true, nil
}
