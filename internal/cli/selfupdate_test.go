package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func buildArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		tw.Write([]byte(body))
	}
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

func TestReleaseAssetNamesMatchGoreleaser(t *testing.T) {
	if got := releaseAssetName("v0.16.0", "darwin", "arm64"); got != "openbotkit_0.16.0_darwin_arm64.tar.gz" {
		t.Errorf("asset = %q", got)
	}
	if got := checksumsAssetName("v0.16.0"); got != "openbotkit_0.16.0_checksums.txt" {
		t.Errorf("checksums = %q", got)
	}
}

func TestVerifyChecksum(t *testing.T) {
	archive := buildArchive(t, map[string]string{"obk": "new binary"})
	sum := sha256.Sum256(archive)
	name := "openbotkit_0.16.0_darwin_arm64.tar.gz"
	checksums := []byte("deadbeef  openbotkit_0.16.0_linux_amd64.tar.gz\n" + hex.EncodeToString(sum[:]) + "  " + name + "\n")

	if err := verifyChecksum(archive, name, checksums); err != nil {
		t.Fatalf("valid archive rejected: %v", err)
	}
	tampered := append([]byte{}, archive...)
	tampered[len(tampered)-1] ^= 0xff
	if err := verifyChecksum(tampered, name, checksums); err == nil {
		t.Fatal("tampered archive accepted")
	}
	if err := verifyChecksum(archive, "openbotkit_0.16.0_linux_arm64.tar.gz", checksums); err == nil {
		t.Fatal("archive without a listed checksum accepted")
	}
}

func TestExtractBinaryAndReplaceThroughSymlink(t *testing.T) {
	archive := buildArchive(t, map[string]string{"README.md": "docs", "obk": "new binary"})
	bin, err := extractBinary(archive, "obk")
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	target := filepath.Join(dir, "obk")
	if err := os.WriteFile(target, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "openbotkit")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	resolved, err := filepath.EvalSymlinks(link)
	if err != nil {
		t.Fatal(err)
	}

	if err := replaceBinary(resolved, bin); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(link)
	if string(got) != "new binary" {
		t.Errorf("binary via symlink = %q", got)
	}
	info, _ := os.Lstat(target)
	if info.Mode().Perm() != 0o755 || !info.Mode().IsRegular() {
		t.Errorf("target mode = %v", info.Mode())
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 2 {
		t.Errorf("leftover temp files: %v", entries)
	}
}

func TestNixManaged(t *testing.T) {
	cases := map[string]bool{
		"/nix/store/9w3k2l1q-openbotkit-0.16.0/bin/obk": true,
		"/Users/me/.local/bin/obk":                      false,
		"/home/me/.local/bin/obk":                       false,
		"/nix/storefront/bin/obk":                       false,
		"/opt/nix/store/abc-openbotkit/bin/obk":         false,
	}
	for path, want := range cases {
		if got := nixManaged(path); got != want {
			t.Errorf("nixManaged(%q) = %v, want %v", path, got, want)
		}
	}
}
